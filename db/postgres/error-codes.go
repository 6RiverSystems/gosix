package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgconn"
)

// reference: https://www.postgresql.org/docs/9.6/errcodes-appendix.html

type PostgreSQLErrorCode string

var (
	SerializationFailure PostgreSQLErrorCode = "40001"
	// InvalidCatalogName often indicates attempting to connect to a database that
	// does not exist
	InvalidCatalogName PostgreSQLErrorCode = "3D000"
	// CannotConnectNow often indicates the server is still starting up
	CannotConnectNow PostgreSQLErrorCode = "57P03"
	DeadlockDetected PostgreSQLErrorCode = "40P01"
)

func IsPostgreSQLErrorCode(err error, code PostgreSQLErrorCode) (*pgconn.PgError, bool) {
	var pgErr *pgconn.PgError
	match := errors.As(err, &pgErr) && pgErr.Code == string(code)
	return pgErr, match
}

func RetryOnErrorCode(code PostgreSQLErrorCode, codes ...PostgreSQLErrorCode) func(context.Context, error) bool {
	allCodes := append([]PostgreSQLErrorCode{code}, codes...)
	return func(ctx context.Context, err error) bool {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			for _, c := range allCodes {
				if pgErr.Code == string(c) {
					return true
				}
			}
		}
		return false
	}
}
