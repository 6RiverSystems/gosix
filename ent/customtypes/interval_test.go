package customtypes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParsePostgreSQLInterval(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantResult time.Duration
		assertion  assert.ErrorAssertionFunc
	}{
		{
			"hh:mm:ss",
			"01:02:03",
			1*time.Hour + 2*time.Minute + 3*time.Second,
			assert.NoError,
		},
		{
			"hh:mm:ss.s",
			"01:02:03.4",
			1*time.Hour + 2*time.Minute + 3*time.Second + 400*time.Millisecond,
			assert.NoError,
		},
		{
			"hh:mm:ss.sssssssss",
			"00:00:00.123456789",
			123456789 * time.Nanosecond,
			assert.NoError,
		},
		{
			"y m d hh:mm:ss.ssssss",
			"1 year 2 mons 3 days 08:09:10.123456",
			365*24*time.Hour + 2*30*24*time.Hour + 3*24*time.Hour + 8*time.Hour + 9*time.Minute + 10*time.Second + 123456*time.Microsecond,
			assert.NoError,
		},
		{
			"err hh:mm:ss.sssssssssS",
			"00:00:00.0000000009",
			time.Duration(0),
			assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := ParsePostgreSQLInterval(tt.input)
			tt.assertion(t, err)
			assert.Equal(t, tt.wantResult, gotResult)
		})
	}
}

func TestInterval_Scan(t *testing.T) {
	tests := []struct {
		name            string
		src             interface{}
		errAssertion    assert.ErrorAssertionFunc
		resultAssertion func(t *testing.T, i Interval)
	}{
		// TODO: Duration
		// TODO: *Duration
		// TODO: int64
		// TODO: *in64
		{
			"string hms",
			"1h0m0s",
			assert.NoError,
			func(t *testing.T, i Interval) {
				assert.Equal(t, time.Hour, time.Duration(i))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var i Interval
			if tt.errAssertion(t, i.Scan(tt.src)) {
				tt.resultAssertion(t, i)
			}
		})
	}
}
