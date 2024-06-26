package main

import (
	openapi "actdata/apis"
	"actdata/models"
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

func MakeAddMeasurement(ctx context.Context, db boil.ContextExecutor) AddMeasurement {
	return func(msrmt openapi.MeasurementView, sid uint64) error {
		m := models.Measurement{
			SessionID: int64(sid),
			Timestamp: msrmt.Timestamp,
			Heartbeat: float64(msrmt.Heartbeat),
		}
		return m.Insert(ctx, db, boil.Infer())
	}
}
func MakeGetMeasurements(ctx context.Context, db boil.ContextExecutor) GetMeasurements {
	return func(session_id uint64) (ms []openapi.MeasurementView, err error) {
		ms_models, err := models.Measurements(
			models.MeasurementWhere.SessionID.EQ(int64(session_id)),
		).All(ctx, db)
		if err != nil {
			return nil, err
		}
		ms = make([]openapi.MeasurementView, len(ms_models))
		for k, v := range ms_models {
			ms[k] = *openapi.NewMeasurementView(v.Timestamp, float32(v.Heartbeat))
		}
		return
	}
}
func MakeGetSession(ctx context.Context, db boil.ContextExecutor) GetSession {
	return func(id uint64) (sess Session, err error) {
		s, err := models.FindSession(ctx, db, int64(id))
		if err != nil {
			err = errorConvert(err)
			return
		}
		return session(*s), nil
	}
}
func MakeGetSessionFromPolar(ctx context.Context, db boil.ContextExecutor) GetSessionFromPolar {
	return func(polar_id uint64) (sess Session, err error) {
		s, err := models.Sessions(
			models.SessionWhere.SessionID.EQ(int64(polar_id)),
		).One(ctx, db)
		if err != nil {
			err = errorConvert(err)
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
func MakeSetSessionToken(ctx context.Context, db boil.ContextExecutor) SetSessionToken {
	return func(sess Session, tok string) error {
		m := models.Session{AuthToken: tok, SessionID: int64(sess.GetID())}
		_, err := m.Update(
			ctx,
			db,
			boil.Whitelist(models.SessionColumns.AuthToken),
		)
		return err

	}
}

func MakeDeleteSession(ctx context.Context, db boil.ContextExecutor) DeleteSession {
	return func(id uint64) error {
		c, err := models.
			Sessions(models.SessionWhere.SessionID.EQ(int64(id))).
			DeleteAll(ctx, db)
		if c == 0 {
			err = errors.Join(err, fiber.ErrNotFound)
		}
		return err
	}
}

// Session impl from DB model
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
