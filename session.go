package main

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type Session interface {
	PolarToken() string
	ID() uint64
	PolarID() uint64
}

type NewSession func(tok string, polar_id uint64) (sess Session, err error)
type GetSession func(id uint64) (Session, error)
type NewSessionToken func(sess Session) (t string, err error)

func MakeNewSessionToken(key interface{}) NewSessionToken {
	return func(sess Session) (t string, err error) {
		tok := jwt.New(jwt.SigningMethodHS256)
		c := tok.Claims.(jwt.MapClaims)
		c["sub"] = fmt.Sprint(sess.ID())
		return tok.SignedString(key)
	}
}

type session struct {
	token   string
	id      uint64
	polarID uint64
}

// type session models.Session {
// }
func (this session) PolarID() uint64 {
	return this.polarID
}

func (this session) PolarToken() string {
	return this.token
}
func (this session) ID() uint64 {
	return this.id
}
