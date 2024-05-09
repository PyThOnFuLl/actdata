package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type Session interface {
	Token() string
	ID() uint64
	PolarID() uint64
}
type RetrieveSession func(c *fiber.Ctx) (sess Session, err error)
type NewSession func(tok string, polar_id uint64) (sess Session, err error)
type NewSessionToken func(sess Session) *jwt.Token
