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
	"testing"

	"github.com/stretchr/testify/assert"

	"go.6river.tech/gosix/migrate"
)

func TestEventMigrationsFor(t *testing.T) {
	type args struct {
		schema string
		table  string
	}
	type sqlExpectation struct {
		name      string
		hasUp     bool
		hasDown   bool
		upCheck   func(*testing.T, string)
		downCheck func(*testing.T, string)
	}
	tests := []struct {
		name string
		args args
		want []sqlExpectation
	}{
		{
			"without schema",
			args{
				schema: "",
				table:  "events_without_schema",
			},
			[]sqlExpectation{
				{
					"0001_base",
					true,
					false,
					func(t *testing.T, contents string) {
						assert.Contains(t, contents, " events_without_schema ")
						assert.NotContains(t, contents, ".events_without_schema")
					},
					nil,
				},
			},
		},
		{
			"with schema",
			args{
				schema: "withschema",
				table:  "events_with_schema",
			},
			[]sqlExpectation{
				{
					"0001_base",
					true,
					false,
					func(t *testing.T, contents string) {
						assert.Contains(t, contents, " withschema.events_with_schema ")
						assert.NotContains(t, contents, " events_with_schema")
					},
					nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EventMigrationsFor(tt.args.schema, tt.args.table)
			assert.Len(t, got, len(tt.want))
			for i := 0; i < len(got) && i < len(tt.want); i++ {
				assert.Equal(t, tt.want[i].name, got[i].Name())
				if assert.IsType(t, (*migrate.SQLMigration)(nil), got[i]) {
					sm := got[i].(*migrate.SQLMigration)
					assertSQL(t, sm, migrate.DirectionUp, tt.want[i].hasUp, tt.want[i].upCheck)
					assertSQL(t, sm, migrate.DirectionDown, tt.want[i].hasDown, tt.want[i].downCheck)
				}
			}
		})
	}
}

func assertSQL(
	t *testing.T,
	m *migrate.SQLMigration,
	dir migrate.Direction,
	want bool,
	contentCheck func(*testing.T, string),
) {
	contents, err := m.Contents(dir)
	// don't expect errors when there's no script, just an empty string
	assert.NoError(t, err)
	var contentOK bool
	if want {
		contentOK = assert.NotEmpty(t, contents)
	} else {
		contentOK = assert.Empty(t, contents)
	}
	if contentOK && contentCheck != nil {
		contentCheck(t, contents)
	}
}
