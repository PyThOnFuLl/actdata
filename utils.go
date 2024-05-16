package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
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
