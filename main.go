package main

import (
	openapi "actdata/apis"
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/golang-jwt/jwt/v5"
	_ "modernc.org/sqlite"
)

func main() {
	fmt.Printf("%+v", f())
}
func f() error {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "./database.db")
	if err != nil {
		return err
	}
	prefix := "/proxy"
	secret := []byte(os.Getenv("TOKEN_SECRET"))
	getS := MakeGetSession(ctx, db)
	retrieveS := MakeRetrieveSession(getS, secret)
	// endpoints
	app := fiber.New()
	app.Get("/oauth2_callback", MakeOauthCallback(
		MakeCode2Token(
			os.Getenv("CLIENT_ID"),
			os.Getenv("CLIENT_SECRET"),
		),
		MakeNewSessionToken(secret),
		MakeNewSession(ctx, db),
	))
	app.Get("/measurements", MakeGetMeasurementsHandler(MakeGetMeasurements(ctx, db), retrieveS))
	app.Get("/info", MakeSessionInfo(retrieveS))
	app.Post("/measurements", MakePostMeasurement(MakeAddMeasurement(ctx, db), retrieveS))
	app.Use(prefix, MakeProxy(prefix, retrieveS))
	return app.Listen(":8000")
}

// получить информацию о сессии
func MakeSessionInfo(rs RetrieveSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := rs(c)
		if err != nil {
			return err
		}
		return c.JSON(openapi.NewSessionView(int64(sess.GetPolarID())))
	}
}

// обработчик запросов, проксирующий их на API AccessLink
func MakeProxy(prefix string, rs RetrieveSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := rs(c)
		if err != nil {
			return err
		}
		req, err := http.NewRequest(
			http.MethodGet,
			"https://www.polaraccesslink.com/v3"+strings.TrimPrefix(string(c.Request().URI().Path()), prefix),
			nil,
		)
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", "Bearer "+sess.GetPolarToken())
		jsonize(req)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		c.Status(resp.StatusCode)
		_, err = io.Copy(c, resp.Body)
		return err
	}
}

// добавить новое измерение ЧСС текущей сессии
type AddMeasurement func(msrmt openapi.MeasurementView, sid uint64) error

func MakePostMeasurement(add AddMeasurement, rs RetrieveSession) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var in openapi.MeasurementView
		if err := c.BodyParser(&in); err != nil {
			return err
		}
		sess, err := rs(c)
		if err != nil {
			return err
		}
		if err := add(in, sess.GetID()); err != nil {
			return err
		}
		return nil
	}
}

// получить все измерения ЧСС текущей сессии
type GetMeasurements func(session_id uint64) ([]openapi.MeasurementView, error)

func MakeGetMeasurementsHandler(gm GetMeasurements, rs RetrieveSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ses, err := rs(c)
		ms, err := gm(ses.GetID())
		if err != nil {
			return err
		}
		return c.JSON(ms)
	}
}

// response from polar token endpoint
type AccessToken struct {
	Value     string `json:"access_token"`
	Type      string `json:"token_type"`
	ExpiresIn uint   `json:"expires_in"`
	XUserID   uint64 `json:"x_user_id"`
}

// обменять код с oauth на токен пользователя
type Code2Token func(code string) (at AccessToken, err error)

func MakeCode2Token(cli_id, cli_secret string) Code2Token {
	return func(code string) (at AccessToken, err error) {
		var bodybuf bytes.Buffer
		bs, err := json.Marshal(map[string]interface{}{
			"grant_type": "authorization_code",
			"code":       code,
		})
		if err != nil {
			return
		}
		_, err = bodybuf.Write(bs)
		if err != nil {
			return
		}
		// http form
		vals := url.Values{}
		vals.Add("grant_type", "authorization_code")
		vals.Add("code", code)
		req, err := http.NewRequest(
			http.MethodPost,
			"https://polarremote.com/v2/oauth2/token",
			strings.NewReader(vals.Encode()),
		)
		if err != nil {
			return
		}
		auth := base64.StdEncoding.EncodeToString([]byte(cli_id + ":" + cli_secret))
		req.Header.Set(
			"Authorization",
			"Basic "+auth,
		)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := http.DefaultClient.Do(req)
		err = json.NewDecoder(resp.Body).Decode(&at)
		return
	}
}
func MakeOauthCallback(
	c2t Code2Token,
	mkTok NewSessionToken,
	newSession NewSession,
) fiber.Handler { // {{{
	return func(c *fiber.Ctx) error {
		code := c.Query("code")
		if code == "" {
			return fiber.ErrBadRequest
		}
		tk, err := c2t(code)
		if err != nil {
			return err
		}
		sess, err := newSession(tk.Value, tk.XUserID)
		if err != nil {
			return err
		}
		r, w := io.Pipe()
		defer r.Close()
		body := (map[string]interface{}{
			"member-id": fmt.Sprint(sess.GetID()),
		})
		enc_err := make(chan error)
		go func() {
			defer w.Close()
			if err := json.NewEncoder(w).Encode(body); err != nil {
				enc_err <- err
				return
			}
			enc_err <- nil
		}()
		req, err := http.NewRequest(http.MethodPost, "https://www.polaraccesslink.com/v3/users", r)
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", "Bearer "+tk.Value)
		jsonize(req)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		if err := <-enc_err; err != nil {
			return err
		}
		defer resp.Body.Close()
		if (resp.StatusCode < 200 || resp.StatusCode >= 300) &&
			resp.StatusCode != http.StatusConflict {
			c.Status(resp.StatusCode)
			return nil
		}
		tok, err := mkTok(sess)
		if err != nil {
			return err
		}
		return c.JSON(tok)
	}
} // }}}

// получить объект Session из контекста запроса
type RetrieveSession func(c *fiber.Ctx) (sess Session, err error)

func MakeRetrieveSession(gs GetSession, secret []byte) RetrieveSession {
	return func(c *fiber.Ctx) (sess Session, err error) {
		auth := c.Request().Header.Peek("Authorization")
		tok, err := jwt.Parse(
			strings.TrimSpace(strings.TrimPrefix(string(auth), "Bearer")),
			func(t *jwt.Token) (interface{}, error) { return []byte(secret), nil },
		)
		if err != nil {
			return
		}
		sub, err := tok.Claims.GetSubject()
		if err != nil {
			return
		}
		id, err := parseUint(sub)
		if err != nil {
			return
		}
		return gs(id)
	}
}
