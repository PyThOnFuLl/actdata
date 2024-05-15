package main

import (
	"actdata/models"
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
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"github.com/golang-jwt/jwt/v5"
	_ "modernc.org/sqlite"
)

const polarflow = "https://flow.polar.com"

func main() {
	fmt.Printf("%+v", f())
}
func f() error {
	app := fiber.New()
	ctx := context.Background()
	db, err := sql.Open("sqlite", "./database.db")
	if err != nil {
		return err
	}
	prefix := "/proxy"
	oauth2_callback := "/oauth2_callback"
	redirect_url := "http://localhost:8000" + oauth2_callback
	secret := []byte(os.Getenv("TOKEN_SECRET"))
	getS := MakeGetSession(ctx, db)
	retrieveS := MakeRetrieveSession(getS, secret)
	app.Get(oauth2_callback, MakeOauthCallback(
		MakeCode2Token(
			os.Getenv("CLIENT_ID"),
			os.Getenv("CLIENT_SECRET"),
		),
		MakeNewSessionToken(secret),
		MakeNewSession(ctx, db),
	))
	app.Get("/measurements", MakeMeasurements(ctx, db, retrieveS))
	app.Use(prefix, MakeProxy(prefix, retrieveS))
	fmt.Println(redirect_url)
	return app.Listen(":8000")
}

type GetAuthToken func(string) (string, error)

func MakeMeasurements(ctx context.Context, db boil.ContextExecutor, rs RetrieveSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ses, err := rs(c)
		if err != nil {
			return err
		}
		ms, err := models.Measurements(qm.Where("session_id = ?", ses.ID())).All(ctx, db)
		if err != nil {
			return err
		}
		return c.JSON(ms)
	}
}

type AccessToken struct {
	Value     string `json:"access_token"`
	Type      string `json:"token_type"`
	ExpiresIn uint   `json:"expires_in"`
	XUserID   uint64 `json:"x_user_id"`
}
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
		fmt.Printf("cli_id: %v\n", cli_id)
		fmt.Printf("cli_secret: %v\n", cli_secret)
		auth := base64.StdEncoding.EncodeToString([]byte(cli_id + ":" + cli_secret))
		fmt.Printf("auth: %v\n", auth)
		req.Header.Set(
			"Authorization",
			"Basic "+auth,
		)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := http.DefaultClient.Do(req)
		fmt.Printf("resp.StatusCode: %v\n", resp.StatusCode)
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
		bs, err := json.Marshal(map[string]interface{}{
			"member-id": fmt.Sprint(sess.ID()),
		})
		if err != nil {
			return err
		}
		body := bytes.Buffer{}
		_, err = body.Write(bs)
		if err != nil {
			return err
		}

		req, err := http.NewRequest(http.MethodPost, "https://www.polaraccesslink.com/v3/users", &body)
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", "Bearer "+tk.Value)
		jsonize(req)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
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
		id, err := strconv.ParseUint(sub, 10, 64)
		if err != nil {
			return
		}
		return gs(id)
	}
}
func MakeGetUserInfo(rs RetrieveSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := rs(c)
		if err != nil {
			return err
		}
		req, err := http.NewRequest(
			http.MethodGet,
			"https://www.polaraccesslink.com/v3/users/"+fmt.Sprint(sess.PolarID()),
			nil,
		)
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", "Bearer "+sess.PolarToken())
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
		req.Header.Add("Authorization", "Bearer "+sess.PolarToken())
		fmt.Printf("req.Header: %+v\n", req.Header)
		fmt.Printf("req.URL.String(): %v\n", req.URL.String())
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
