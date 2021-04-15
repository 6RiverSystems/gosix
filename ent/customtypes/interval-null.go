package customtypes

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"entgo.io/ent/schema/field"
)

type NullInterval struct {
	Valid    bool
	Interval Interval
}

var _ driver.Valuer = NullInterval{}
var _ field.ValueScanner = (*NullInterval)(nil)
var _ json.Marshaler = NullInterval{}
var _ json.Unmarshaler = (*NullInterval)(nil)

func (i NullInterval) Value() (driver.Value, error) {
	if !i.Valid {
		return nil, nil
	}
	// PostgreSQL understands Go's duration string format
	return i.Interval.String(), nil
}

func (i *NullInterval) Scan(src interface{}) error {
	if src == nil {
		i.Valid = false
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
		return fmt.Errorf("Unsupported scan type: %T", src)
	}
	i.Valid = true
	i.Interval = Interval(d)
	return nil
}

func (i *NullInterval) NotNull() bool {
	return i != nil && i.Valid
}

func (i NullInterval) MarshalJSON() ([]byte, error) {
	if !i.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(i.Interval)
}

func (i *NullInterval) UnmarshalJSON(data []byte) error {
	var s *string
	var err error
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == nil {
		i.Valid = false
		return nil
	}
	d, err := ParsePostgreSQLInterval(*s)
	if err != nil {
		return err
	}
	i.Interval = Interval(d)
	return nil
}
