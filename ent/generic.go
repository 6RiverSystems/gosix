package ent

import (
	"context"
	"database/sql"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql/schema"
)

// FUTURE: it'd be nice if ent itself provided these common-baseline interfaces

type EntTx interface {
	Commit() error
	Rollback() error
}

type EntClient interface {
	Close() error

	GetSchema() EntClientSchema

	BeginTxGeneric(ctx context.Context, opts *sql.TxOptions) (EntTx, error)

	EntityClient(string) EntityClient
}

type EntityClient interface {
	CreateEntity() EntityCreate
}

type EntityCreate interface {
	EntityMutation() ent.Mutation
	SaveEntity(context.Context) (interface{}, error)
}

// Generic ent-style Schema (ent doesn't provide an interface for this)
type EntClientSchema interface {
	Create(ctx context.Context, opts ...schema.MigrateOption) error
}
