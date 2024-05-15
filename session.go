package main

import (
	"actdata/models"
	"context"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

type Session interface {
	PolarToken() string
	ID() uint64
	PolarID() uint64
}

type GetSession func(id uint64) (Session, error)

type NewSession func(tok string, polar_id uint64) (sess Session, err error)

func MakeGetSession(ctx context.Context, db boil.ContextExecutor) GetSession {
	return func(id uint64) (sess Session, err error) {
		s, err := models.FindSession(ctx, db, int64(id))
		if err != nil {
			return
		}
		return session{token: s.AuthToken, id: uint64(s.SessionID), polarID: s.PolarID}, nil
	}
}
func MakeNewSession(ctx context.Context, db boil.ContextExecutor) NewSession {
	return func(tok string, polar_id uint64) (sess Session, err error) {
		m := models.Session{
			PolarID:   polar_id,
			AuthToken: tok,
		}
		if err = m.Insert(ctx, db, boil.Infer()); err != nil {
			return
		}
		return session{
			id:      uint64(m.SessionID),
			polarID: m.PolarID,
			token:   m.AuthToken,
		}, nil
	}
}

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
