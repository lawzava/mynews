package main

import (
	"crypto/md5" // nolint:gosec speed is higher concern than security in this use case
	"encoding/hex"
	"strings"
)

func buildStoryID(in ...string) string {
	h := md5.New() // nolint:gosec speed is higher concern than security in this use case

	_, _ = h.Write([]byte(strings.Join(in, "-")))

	return hex.EncodeToString(h.Sum(nil))
}
