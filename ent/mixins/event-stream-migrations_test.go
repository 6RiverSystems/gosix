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
