package main

import (
	"actdata/models"
	"context"

	"github.com/volatiletech/sqlboiler/v4/boil"
)

func MakeGetSession(ctx context.Context, db boil.ContextExecutor) GetSession {
	return func(id uint64) (sess Session, err error) {
		s, err := models.FindSession(ctx, db, int64(id))
		if err != nil {
			return
		}
		return session(*s), nil
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
		return session(m), nil
	}
}

type session models.Session

func (this session) GetPolarID() uint64 {
	return this.PolarID
}

func (this session) GetPolarToken() string {
	return this.AuthToken
}
func (this session) GetID() uint64 {
	return uint64(this.SessionID)
}
