// Copyright (c) 2021 6 River Systems
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package ent

import (
	"context"
	"database/sql"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql/schema"
)

// FUTURE: it'd be nice if ent itself provided these common-baseline interfaces

type EntTx[C EntClient] interface {
	Commit() error
	Rollback() error
	Client() C
}

type EntClient interface {
	Close() error

	GetSchema() EntClientSchema

	BeginTxGeneric(ctx context.Context, opts *sql.TxOptions) (EntTx[EntClient], error)

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
