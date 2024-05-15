package main

import (
	"net/http"
	"strconv"
)

func jsonize(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}
func parseUint(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}
