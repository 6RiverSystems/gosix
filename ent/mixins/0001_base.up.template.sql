-- Copyright (c) 2021 6 River Systems
--
-- Permission is hereby granted, free of charge, to any person obtaining a copy of
-- this software and associated documentation files (the "Software"), to deal in
-- the Software without restriction, including without limitation the rights to
-- use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
-- the Software, and to permit persons to whom the Software is furnished to do so,
-- subject to the following conditions:
--
-- The above copyright notice and this permission notice shall be included in all
-- copies or substantial portions of the Software.
--
-- THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
-- IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
-- FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
-- COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
-- IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
-- CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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
