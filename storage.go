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
