package oas

import (
	"fmt"
	"time"
)

type Time struct {
	time.Time
}

const OASRFC3339Millis = "2006-01-02T15:04:05.000Z07:00"

func Now() Time {
	return Time{time.Now().UTC().Truncate(time.Millisecond)}
}

func FromTime(value time.Time) Time {
	return Time{value.UTC().Truncate(time.Millisecond)}
}

func (t Time) MarshalJSON() ([]byte, error) {
	// TODO: pre-allocate to make it faster
	return []byte(fmt.Sprintf(`"%s"`, t.UTC().Format(OASRFC3339Millis))), nil
}
