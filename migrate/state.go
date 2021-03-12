package migrate

import "time"

// struct State represents an entry in the migrations table, structured to
// be compatible with nodejs db-migrate
type State struct {
	ID    int64     `db:"id"`
	Name  string    `db:"name"`
	RunOn time.Time `db:"run_on"`
}
