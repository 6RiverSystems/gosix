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
