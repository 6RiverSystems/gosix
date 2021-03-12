package db

import (
	"context"
	"io/fs"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"go.6river.tech/gosix/ent"
	"go.6river.tech/gosix/migrate"
)

func MigrateUp(
	ctx context.Context,
	// dirty hack, ORM reference, currently must be a *ent.Client
	migrateVia interface{},
	migrationsFS fs.FS,
) error {
	migrations, err := migrate.LoadFS(migrationsFS, nil)
	if err != nil {
		return err
	}

	m := (&migrate.Migrator{}).AddAndSort("", migrations...)

	return Up(ctx, migrateVia, m)
}

func Up(
	ctx context.Context,
	// dirty hack, ORM reference, currently must be a *ent.Client
	migrateVia interface{},
	migrator *migrate.Migrator,
) error {
	sql, driverName, dialectName, err := OpenDefault()
	if sql != nil {
		// TODO: error check
		defer sql.Close()
	}
	if err != nil {
		return errors.Wrap(err, "Failed to connect to DB for Up migration")
	}

	if !migrator.HasMigrations() || dialectName == SqliteDialect {
		switch mdb := migrateVia.(type) {
		case ent.EntClient:
			err := MigrateUpEnt(ctx, mdb.GetSchema())
			if err != nil {
				return errors.Wrapf(err, "Failed Up migration via ent for %s", dialectName)
			}
			return nil
		default:
			return errors.Errorf("Unrecognized migrateVia for %s: %T", dialectName, migrateVia)
		}
	}

	switch dialectName {
	case PostgresDialect:
		migrator = migrator.WithDialect(&migrate.PgxDialect{})
	default:
		// TODO: if we had a dialect registry could make this generic
		return errors.Errorf("Unrecognized dialect '%s'", dialectName)
	}

	// default fallthrough assumes we've initialized migrator with a dialect
	return migrator.Up(ctx, sqlx.NewDb(sql, driverName))
}
