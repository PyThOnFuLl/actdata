package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func jsonize(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}
func lookupEnv(k string) (string, error) {
	v, ok := os.LookupEnv(k)
	if !ok {
		return v, fmt.Errorf("%s environment variable is not defined", k)
	}
	return v, nil
}
func parseUint(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}
func errorConvert(err error) error {
	var fi error
	if errors.Is(err, sql.ErrNoRows) {
		fi = fiber.ErrNotFound
	}
	return errors.Join(err, fi)
}
