package mixins

import (
	"embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"go.6river.tech/gosix/migrate"
)

// TODO: use go:embed to source the templates here
// TODO: and/or embed up & down as named templates to be invoked

//go:embed *.sql
var migrationsFS embed.FS

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
	entries, err := migrationsFS.ReadDir(".")
	if err != nil {
		// this should never happen
		panic(errors.Wrap(err, "Unable to list embedded migrations"))
	}
	ret := make([]migrate.Migration, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		baseName := strings.TrimSuffix(name, ".template.sql")
		if baseName == name {
			// didn't have the suffix
			continue
		}
		migrationName := strings.TrimSuffix(baseName, ".up")
		if migrationName == baseName {
			panic(errors.Errorf("Unexpected non-up migration found in embedded migrations: %s", baseName))
		}
		ret = append(ret, migrate.FromContent(
			migrationName,
			func() (string, error) {
				content, err := migrationsFS.ReadFile(name)
				if err != nil {
					return "", errors.Wrapf(err, "Unable to read embedded migration %s", name)
				}
				t := template.New(baseName)
				_, err = t.Parse(string(content))
				if err != nil {
					return "", err
				}
				buf := &strings.Builder{}
				err = t.Execute(buf, eventStreamParams{Schema: schema, Table: table})
				return buf.String(), err
			},
			nil,
		))
	}
	return ret
}
