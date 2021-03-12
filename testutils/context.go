package testutils

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type testContextKey string

var testFromContextKey testContextKey = testContextKey(uuid.NewString())

func ContextForTest(t testing.TB) context.Context {
	ctx := context.Background()
	if tt, ok := t.(*testing.T); ok {
		if d, ok := tt.Deadline(); ok {
			var cancel func()
			ctx, cancel = context.WithDeadline(ctx, d)
			ctx = context.WithValue(ctx, testFromContextKey, t)
			t.Cleanup(cancel)
		}
	}
	return ctx
}

func DeadlineForTest(t testing.TB) time.Time {
	if tt, ok := t.(*testing.T); ok {
		if deadline, ok := tt.Deadline(); ok {
			return deadline
		}
	}
	return time.Now().Add(time.Minute)
}

func TestForContext(ctx context.Context) testing.TB {
	return ctx.Value(testFromContextKey).(testing.TB)
}
