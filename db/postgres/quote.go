package postgres

import (
	"encoding/hex"
	"strings"
)

// copy-pasta from pgx internals, see
// https://github.com/jackc/pgx/blob/master/conn.go#L341 and
// https://github.com/jackc/pgx/issues/202
func QuoteIdentifier(s string) string {
	return `"` + strings.Replace(s, `"`, `""`, -1) + `"`
}

// copy-pasta from pgx internals, see
// https://github.com/jackc/pgx/blob/master/internal/sanitize/sanitize.go
func QuoteString(str string) string {
	return "'" + strings.Replace(str, "'", "''", -1) + "'"
}

// copy-pasta from pgx internals, see
// https://github.com/jackc/pgx/blob/master/internal/sanitize/sanitize.go
func QuoteBytes(buf []byte) string {
	return `'\x` + hex.EncodeToString(buf) + "'"
}
