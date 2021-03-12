package mixins

import (
	"fmt"
	"strings"
	"text/template"

	"go.6river.tech/gosix/migrate"
)

// TODO: use go:embed to source the templates here
// TODO: and/or embed up & down as named templates to be invoked

const up0001 = `
CREATE TABLE {{.SchemaQualifiedTable}} (
	id uuid not null primary key,
	persisted_at timestamptz not null default now(),
	sequence_id bigserial not null unique,
	event_type text not null,
	scope_type text not null,
	scope_id text not null,
	happened_at timestamptz not null,
	data jsonb null
);
CREATE INDEX ix_{{.Table}}_scope_happened on {{.SchemaQualifiedTable}} (
	scope_type,
	scope_id,
	happened_at
);
CREATE INDEX ix_{{.Table}}_scope_ingested on {{.SchemaQualifiedTable}} (
	scope_type,
	scope_id,
	sequence_id
);
CREATE INDEX ix_{{.Table}}_scope_type_happened_at on {{.SchemaQualifiedTable}} (
	scope_type,
	happened_at
);

CREATE INDEX ix_{{.Table}}_persisted_at ON {{.SchemaQualifiedTable}} (
	persisted_at
);

create index ix_{{.Table}}_data_path_ops
on {{.SchemaQualifiedTable}}
using gin (data jsonb_path_ops);
`

type eventStreamParams struct {
	Schema string
	Table  string
}

func (p eventStreamParams) SchemaQualifiedTable() string {
	if p.Schema == "" {
		return p.Table
	}
	return fmt.Sprintf("%s.%s", p.Schema, p.Table)
}

// EventMigrationsFor returns a set of migrations to get the current event
// stream table schema. You should wrap this in WithPrefix to assign a unique
// prefix to these migrations for each event stream.
func EventMigrationsFor(schema, table string) []migrate.Migration {
	return []migrate.Migration{
		migrate.FromContent(
			"0001_base",
			func() (string, error) {
				t := template.New("0001_base.up")
				_, err := t.Parse(up0001)
				if err != nil {
					return "", err
				}
				buf := &strings.Builder{}
				err = t.Execute(buf, eventStreamParams{Schema: schema, Table: table})
				return buf.String(), err
			},
			nil,
		),
	}
}
