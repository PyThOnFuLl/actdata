package main

import (
	openapi "actdata/apis"
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"

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
	proxy_prefix := "/proxy"
	fh_prefix := "/fasthttp"
	secret := []byte(os.Getenv("TOKEN_SECRET"))
	getS := MakeGetSession(ctx, db)
	retrieveS := MakeRetrieveSession(getS, secret)
	proxy := MakeProxy()
	fh_proxy := MakeFasthttpProxy(&fasthttp.Client{})
	// endpoints
	app := fiber.New()
	app.Get("/oauth2_callback", MakeOauthCallback(
		MakeCode2Token(
			os.Getenv("CLIENT_ID"),
			os.Getenv("CLIENT_SECRET"),
		),
		MakeNewSessionToken(secret),
		MakeSetSessionToken(ctx, db),
		MakeRegisterUser(proxy),
		MakeGetSessionFromPolar(ctx, db),
		MakeNewSession(ctx, db),
	))
	app.Get("/measurements", MakeGetMeasurementsHandler(MakeGetMeasurements(ctx, db), retrieveS))
	app.Get("/info", MakeSessionInfo(retrieveS))
	app.Post("/measurements", MakePostMeasurement(MakeAddMeasurement(ctx, db), retrieveS))
	app.Use(proxy_prefix, MakeProxyHandler(proxy_prefix, retrieveS, proxy))
	app.Use(fh_prefix, MakeProxyHandler(fh_prefix, retrieveS, fh_proxy))
	return app.Listen(":8000")
}

// получить информацию о сессии
func MakeSessionInfo(rs RetrieveSession) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := rs(c)
		if err != nil {
			return err
		}
		return c.JSON(openapi.NewSessionView(int64(sess.GetPolarID()), int64(sess.GetID())))
	}
}

type Proxy func(token, endpoint, method string, reqbody io.Reader, header *fasthttp.RequestHeader) (status int, respbody io.ReadCloser, err error)

func MakeFasthttpProxy(c *fasthttp.Client) Proxy {
	return func(token, endpoint, method string, reqbody io.Reader, header *fasthttp.RequestHeader) (status int, respbody io.ReadCloser, err error) {
		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)
		req.SetBodyStream(respbody, -1)
		header.CopyTo(&req.Header)
		req.Header.SetMethod(method)
		req.Header.Set("Authorization", "Bearer "+token)
		req.SetRequestURI("https://www.polaraccesslink.com/v3" + endpoint)
		resp := fasthttp.AcquireResponse()
		if err = c.Do(req, resp); err != nil {
			return
		}
		return resp.StatusCode(), io.NopCloser(bytes.NewBuffer(resp.Body())), nil
	}
}
func MakeProxy() Proxy {
	return func(token, endpoint, method string, reqbody io.Reader, h *fasthttp.RequestHeader) (status int, respbody io.ReadCloser, err error) {
		proxy_headers := make(http.Header, h.Len())
		h.VisitAll(func(key, value []byte) {
			proxy_headers.Add(string(key), string(value))

		})
		req, err := http.NewRequest(method, "https://www.polaraccesslink.com/v3"+endpoint, reqbody)
		if err != nil {
			return
		}
		req.Header = proxy_headers
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		return resp.StatusCode, resp.Body, err
	}
}

// обработчик запросов, проксирующий их на API AccessLink
func MakeProxyHandler(prefix string, rs RetrieveSession, proxy Proxy) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := rs(c)
		if err != nil {
			return err
		}
		status, body, err := proxy(
			sess.GetPolarToken(),
			strings.TrimPrefix(string(c.Request().URI().Path()), prefix),
			c.Method(),
			c.Request().BodyStream(),
			&c.Request().Header,
		)
		defer body.Close()
		if err != nil {
			return err
		}
		c.Status(status)
		_, err = io.Copy(c, body)
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

// perform registration of user to polar client
type RegisterUser func(sid uint64, token string) error

func MakeRegisterUser(proxy Proxy) RegisterUser {
	return func(sid uint64, token string) error {
		r, w := io.Pipe()
		defer r.Close()
		go func() {
			defer w.Close()
			body := (map[string]interface{}{
				"member-id": fmt.Sprint(sid),
			})
			w.CloseWithError(json.NewEncoder(w).Encode(body))
		}()
		headers := fasthttp.RequestHeader{}
		func(json_header string) {
			headers.Add("content-type", json_header)
			headers.Add("accept", json_header)
		}("application/json")
		status, body, err := proxy(
			token,
			"/users",
			http.MethodPost,
			r,
			&headers,
		)
		if err != nil {
			return err
		}
		defer body.Close()
		if (status < 200 || status >= 300) &&
			status != http.StatusConflict {
			errstr, err := io.ReadAll(body)
			if err != nil {
				fmt.Printf("err reading body: %+v\n", err)
			}
			return fiber.NewError(status, string(errstr))
		}
		return nil
	}
}
func MakeOauthCallback(
	c2t Code2Token,
	mkTok NewSessionToken,
	setTok SetSessionToken,
	reg RegisterUser,
	tryFindSession GetSessionFromPolar,
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
		sess, err := tryFindSession(tk.XUserID)
		if err != nil {
			// if session doesn't exist yet
			if errors.Is(err, fiber.ErrNotFound) {
				// start new session
				sess, err = newSession(tk.Value, tk.XUserID)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			if err := setTok(sess, tk.Value); err != nil {
				return err
			}
		}
		if err := reg(sess.GetID(), tk.Value); err != nil {
			return err
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
