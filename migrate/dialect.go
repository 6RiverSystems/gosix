package migrate

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type Dialect interface {
	DefaultConfig() *Config
	Verify(db *sqlx.DB) error
	EnsureMigrationsTable(context.Context, *sqlx.DB, *Config) error
	SelectStates(*Config) string
	InsertMigration(*Config) string
	DeleteMigration(*Config) string
	BeforeMigration(
		ctx context.Context,
		db *sqlx.DB,
		tx *sqlx.Tx,
		name string,
		direction Direction,
	) error
}
