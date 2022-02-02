//go:build cgo
// +build cgo

package db

import (
	"fmt"
	"net/url"
	"strings"
)

// in CGO mode, use the github.com/mattn/go-sqlite3 driver
const sqliteDriver = "sqlite3"

func SQLiteDSN(filename string, fileScheme, memory bool) string {
	if strings.HasPrefix(filename, "/") {
		// sqlite:relativepath or sqlite:///some/abs/path
		filename = "//" + filename
	}
	if !strings.HasSuffix(filename, ".sqlite3") {
		filename = filename + ".sqlite3"
	}
	scheme := "sqlite"
	if fileScheme {
		scheme = "file"
	}
	// WARNING: keep these in sync with the non-CGO version
	q := url.Values{
		"_fk":           []string{"true"},
		"_journal_mode": []string{"wal"},
		"cache":         []string{"private"},
		"_busy_timeout": []string{"10000"},
		// we need BEGIN IMMEDIATE for several use cases to work
		"_txlock": []string{"immediate"},
	}
	if memory {
		// memory mode needs either shared cache, or single connection. shared cache
		// doesn't play nice and results in lots of unfixable "table is locked"
		// errors, so instead rely on `Open` to set the max conns appropriately
		q.Set("mode", "memory")
		// q.Set("cache", "shared")

		// even so, memory mode is not safe, as a transaction associated with a
		// canceled context will cause the connection to be forcibly closed and thus
		// the whole database to be lost, schema and all:
		// https://github.com/mattn/go-sqlite3/issues/923
	}
	return fmt.Sprintf("%s:%s?%s", scheme, filename, q.Encode())
}
