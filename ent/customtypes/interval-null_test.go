package customtypes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNullInterval_Scan(t *testing.T) {
	tests := []struct {
		name            string
		src             interface{}
		errAssertion    assert.ErrorAssertionFunc
		resultAssertion func(t *testing.T, i NullInterval)
	}{
		{
			"null/nil",
			nil,
			assert.NoError,
			func(t *testing.T, i NullInterval) {
				assert.False(t, i.Valid)
				assert.Equal(t, time.Duration(0), time.Duration(i.Interval))
			},
		},
		// TODO: Duration
		// TODO: *Duration
		// TODO: int64
		// TODO: *in64
		{
			"string hms",
			"1h0m0s",
			assert.NoError,
			func(t *testing.T, i NullInterval) {
				assert.True(t, i.Valid)
				assert.Equal(t, time.Hour, time.Duration(i.Interval))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var i NullInterval
			if tt.errAssertion(t, i.Scan(tt.src)) {
				tt.resultAssertion(t, i)
			}
		})
	}
}
