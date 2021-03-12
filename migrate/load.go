package migrate

import (
	"io/fs"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var nameRegex = regexp.MustCompile(`^(.*)[.-](up|down).sql$`)

func LoadFS(
	migrationsFS fs.FS,
	filter func(*SQLMigration) bool,
) ([]Migration, error) {
	entries := make(map[string]*SQLMigration)
	err := fs.WalkDir(migrationsFS, ".", func(fn string, d fs.DirEntry, err error) error {
		if !d.Type().IsRegular() {
			return nil
		}

		m := nameRegex.FindStringSubmatch(fn)
		if m == nil {
			return errors.Errorf("Entry '%s' does not look like a migration", fn)
		}
		name, direction := m[1], m[2]
		// for compatibility with db-migrate, where things are always scope/name,
		// where scope may be the empty string, so e.g. "a/b" or "/c", but never "c"
		// or "/a/b"
		if strings.IndexByte(name, '/') < 0 {
			name = "/" + name
		}
		e, ok := entries[name]
		if !ok {
			e = &SQLMigration{name: name}
		}
		content := func() (string, error) {
			data, err := fs.ReadFile(migrationsFS, fn)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
		if direction == "up" {
			if e.up != nil {
				return errors.Errorf("Multiple up entries for '%s'", name)
			}
			e.up = content
		} else {
			if e.down != nil {
				return errors.Errorf("Multiple down entries for '%s'", name)
			}
			e.down = content
		}
		entries[name] = e
		return nil
	})
	if err != nil {
		return nil, err
	}
	// make it a slice
	migrations := make([]Migration, 0, len(entries))
	for _, m := range entries {
		if filter == nil || filter(m) {
			migrations = append(migrations, m)
		}
	}
	return migrations, nil
}
