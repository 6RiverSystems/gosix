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

package mixins

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
	"github.com/google/uuid"
)

// EventStream is a mixin that provides the standard schema for the sixriver
// event sourcing data model. While it is possible to add extra fields to this
// schema, you should not do so. Every field in this mixin is declared as
// Immutable as events should never be modified after saving.
type EventStream struct {
	mixin.Schema
}

func (EventStream) Fields() []ent.Field {
	return []ent.Field{
		// id is implicitly the primary key
		// have to assign a default up here because:
		// a) sqlite
		// b) ent bug: https://github.com/ent/ent/issues/781
		field.UUID("id", uuid.UUID{}).StorageKey("id").Default(uuid.New).Immutable(),
		field.Time("persistedAt").StorageKey("persisted_at").Default(utcNow).Immutable(),
		// TODO: while Optional and Nillable are not the same, Optional does equate
		// to Nullable, which causes incorrect generated migrations. But we need it
		// to be optional to permit the db-generated value in Postgres.
		field.Int64("sequenceId").StorageKey("sequence_id").Optional().Immutable().Unique(),
		field.String("eventType").StorageKey("event_type").NotEmpty().Immutable(),
		field.String("scopeType").StorageKey("scope_type").NotEmpty().Immutable(),
		field.String("scopeId").StorageKey("scope_id").NotEmpty().Immutable(),
		field.Time("happenedAt").StorageKey("happened_at").Default(utcNow).Immutable(),
		field.JSON("data", json.RawMessage{}).StorageKey("data").Immutable(),
	}
}

func (EventStream) Indexes() []ent.Index {
	// single column unique indexes/constraints are in the Fields, plain indexes and multi-column unique constraints are here
	return []ent.Index{
		index.Fields("persistedAt").StorageKey("ix_inventory_events_persisted_at"),
		index.Fields("scopeType", "scopeId", "happenedAt").StorageKey("ix_inventory_events_scope_happened"),
		index.Fields("scopeType", "scopeId", "sequenceId").StorageKey("ix_inventory_events_scope_ingested"),
		index.Fields("scopeType", "happenedAt").StorageKey("ix_inventory_events_scope_type_happened_at"),
		// TODO: data jsonb index, can't represent that in ent as it's psql custom syntax
	}
}

func (EventStream) Hooks() []ent.Hook {
	return []ent.Hook{
		// Disallow any ops except create on events
		func(next ent.Mutator) ent.Mutator {
			return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
				if !m.Op().Is(ent.OpCreate) {
					return nil, fmt.Errorf("%s operation is not allowed", m.Op())
				}
				return next.Mutate(ctx, m)
			})
		},
	}
}

func utcNow() time.Time {
	return time.Now().UTC()
}

type EventMutation interface {
	ID() (uuid.UUID, bool)
	SetScopeType(string)
	SetScopeId(string)
	SetEventType(string)
	SetHappenedAt(time.Time)
	SetData(json.RawMessage)
}
