package customtypes

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"entgo.io/ent/schema/field"
)

// TODO: this should be just a direct type, but ent code generation generates
// invalid code for that case, so it has to be a struct
type IntervalNull struct {
	*time.Duration
}

var _ driver.Valuer = Interval{}
var _ field.ValueScanner = (*Interval)(nil)
var _ json.Marshaler = Interval{}
var _ json.Unmarshaler = (*Interval)(nil)

func (i IntervalNull) Value() (driver.Value, error) {
	if i.Duration == nil {
		return nil, nil
	}
	// PostgreSQL understands Go's duration string format
	return i.String(), nil
}

func (i *IntervalNull) Scan(src interface{}) error {
	if src == nil {
		i.Duration = nil
		return nil
	}
	var d time.Duration
	switch s := src.(type) {
	case time.Duration:
		d = s
	case *time.Duration:
		d = *s
	case int64:
		d = time.Duration(s)
	case *int64:
		d = time.Duration(*s)
	case string:
		var err error
		d, err = ParsePostgreSQLInterval(s)
		if err != nil {
			return err
		}
	default:
		return errors.New("Unsupported scan type")
	}
	i.Duration = &d
	return nil
}

func (i *IntervalNull) NotNil() bool {
	return i != nil && i.Duration != nil
}

func (i IntervalNull) MarshalJSON() ([]byte, error) {
	if i.Duration == nil {
		return json.Marshal(nil)
	}
	return json.Marshal(i.String())
}

func (i *IntervalNull) UnmarshalJSON(data []byte) error {
	var s *string
	var err error
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == nil {
		i.Duration = nil
		return nil
	}
	d, err := ParsePostgreSQLInterval(*s)
	if err != nil {
		return err
	}
	i.Duration = &d
	return nil
}
