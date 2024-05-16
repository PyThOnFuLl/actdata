package main

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type Session interface {
	GetPolarToken() string
	GetID() uint64
	GetPolarID() uint64
}

type NewSession func(tok string, polar_id uint64) (sess Session, err error)
type GetSession func(id uint64) (Session, error)
type GetSessionFromPolar func(polar_id uint64) (Session, error)
type NewSessionToken func(sess Session) (t string, err error)
type SetSessionToken func(sess Session, tok string) error

func MakeNewSessionToken(key interface{}) NewSessionToken {
	return func(sess Session) (t string, err error) {
		tok := jwt.New(jwt.SigningMethodHS256)
		c := tok.Claims.(jwt.MapClaims)
		c["sub"] = fmt.Sprint(sess.GetID())
		return tok.SignedString(key)
	}
}
