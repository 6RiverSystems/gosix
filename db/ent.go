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
