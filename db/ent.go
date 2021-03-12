package db

import (
	"context"

	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/schema"

	"go.6river.tech/gosix/ent"
)

// this is mostly a workaround for ent not recognizing pgx
// the problem is this pretty much requires generics
// ... so it doesn't exist
// instead what we have here is just a db uri helper

func OpenSqlForEnt() (*entsql.Driver, error) {
	// compare to client.Open (from the ent generated code)
	if db, _, dialectName, err := OpenDefault(); err != nil {
		return nil, err
	} else {
		return entsql.OpenDB(dialectName, db), nil
	}
}

func MigrateUpEnt(
	ctx context.Context,
	clientSchema ent.EntClientSchema,
) error {
	return clientSchema.Create(ctx, schema.WithDropIndex(true), schema.WithDropColumn(true))
}
