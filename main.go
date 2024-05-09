package main

import (
	"bytes"
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

	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/golang-jwt/jwt/v5"
)

const polarflow = "https://flow.polar.com"

type AccessToken struct {
	Value     string `json:"access_token"`
	Type      string `json:"token_type"`
	ExpiresIn uint   `json:"expires_in"`
	XUserID   uint64 `json:"x_user_id"`
}

func main() {
	app := fiber.New()
	// prefix := "/proxy"
	oauth2_callback := "/oauth2_callback"
	redirect_url := "http://localhost:8000" + oauth2_callback
	// app.Post("/session", MakePostSession())
	app.Get(oauth2_callback, MakeOauthCallback(
		os.Getenv("CLIENT_ID"),
		os.Getenv("CLIENT_SECRET"),
		redirect_url,
	))
	// app.Use(prefix, MakeProxy(prefix, MakeGetAuthToken()))
	fmt.Println(redirect_url)
	app.Listen(":8000")
}

type GetAuthToken func(string) (string, error)

func MakeGetAuthToken() GetAuthToken {
	panic("not implemented")
	// return func(ses string) (string, error) {
	// 	panic("not implemented")
	// }
}

func MakeOauthCallback(
	cli_id,
	cli_secret,
	redirect_url string,
	mkSession NewSession,
	mkTok NewSessionToken,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := c.Query("code")
		if code == "" {
			return fiber.ErrBadRequest
		}
		var body bytes.Buffer
		bs, err := json.Marshal(map[string]interface{}{
			"grant_type": "authorization_code",
			"code":       code,
		})
		if err != nil {
			return err
		}
		_, err = body.Write(bs)
		if err != nil {
			return err
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
			return err
		}
		req.Header.Set(
			"Authorization",
			"Basic "+base64.StdEncoding.EncodeToString([]byte(cli_id+":"+cli_secret)),
		)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := http.DefaultClient.Do(req)
		fmt.Printf("resp.StatusCode: %v\n", resp.StatusCode)
		var tk AccessToken
		if err := json.NewDecoder(resp.Body).Decode(&tk); err != nil {
			return err
		}
		sess, err := mkSession(tk.Value, tk.XUserID)
		if err != nil {
			return err
		}
		bs, err = json.Marshal(map[string]interface{}{
			"member-id": fmt.Sprint(sess.ID()),
		})
		if err != nil {
			return err
		}
		body.Reset()
		_, err = body.Write(bs)
		if err != nil {
			return err
		}

		req, err = http.NewRequest(http.MethodPost, "https://www.polaraccesslink.com/v3/users", &body)
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", "Bearer "+tk.Value)
		jsonize(req)
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if (resp.StatusCode < 200 || resp.StatusCode >= 300) &&
			resp.StatusCode != http.StatusConflict {
			c.Status(resp.StatusCode)
			return nil
		}
		tok := mkTok(sess)
		return c.JSON(tok)
	}
}

type GetSession func(id uint64) (Session, error)

func MakeRetrieveSession(gs GetSession) RetrieveSession {
	return func(c *fiber.Ctx) (sess Session, err error) {
		auth := c.Request().Header.Peek("Authorization")
		tok, err := jwt.Parse(
			string(auth),
			func(t *jwt.Token) (interface{}, error) { return nil, nil },
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
		req.Header.Add("Authorization", "Bearer "+sess.Token())
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
func MakeNewSessionToken(sess Session) NewSessionToken {
	return func() *jwt.Token {
		t := jwt.New(jwt.SigningMethodHS256)
		c := t.Claims.(jwt.MapClaims)
		c["sid"] = sess.ID()
		return t
	}
}
func MakeProxy(prefix string, getAuthToken GetAuthToken) fiber.Handler {
	return func(c *fiber.Ctx) error {
		r := c.Request()
		u := r.URI()
		u.SetPath(strings.TrimPrefix(string(u.Path()), prefix))
		ses := r.Header.Peek("Authorization")
		tok, err := getAuthToken(string(ses))
		if err != nil {
			return err
		}
		r.Header.Set("Authorization", tok)
		if err := proxy.Do(c, polarflow); err != nil {
			return err
		}
		return nil
	}
}
