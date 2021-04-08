package logging

import (
	"sync/atomic"

	"github.com/rs/zerolog"
)

type logBuilder = func() (int32, zerolog.Logger)

// Logger exists to wrap a replaceable zerolog.Logger so levels can be changed dynamically
type Logger struct {
	// c is the current generation value that was active when the logBuilder was
	// last called, to fill the l field
	c int32
	// l is the current underlying logger
	l zerolog.Logger
	// b is the builder function to make an updated logger when the config
	// generation changes
	b logBuilder
}

func newFrom(g logBuilder) *Logger {
	c, l := g()
	return &Logger{c, l, g}
}

// update compares the current configGeneration with the one last used to
// refresh the underlying logger, uses the builder to refresh the logger if it
// has changed, and returns the (new) underlying logger in either case
func (l *Logger) update() *zerolog.Logger {
	cc := atomic.LoadInt32(&configGeneration)
	if cc != l.c {
		l.c, l.l = l.b()
	}
	return &l.l
}

func (l *Logger) Current() zerolog.Logger {
	return *l.update()
}

func (l *Logger) Trace() *zerolog.Event {
	return l.update().Trace()
}
func (l *Logger) Debug() *zerolog.Event {
	return l.update().Debug()
}
func (l *Logger) Info() *zerolog.Event {
	return l.update().Info()
}
func (l *Logger) Warn() *zerolog.Event {
	return l.update().Warn()
}
func (l *Logger) Error() *zerolog.Event {
	return l.update().Error()
}
func (l *Logger) Fatal() *zerolog.Event {
	return l.update().Fatal()
}

func (l *Logger) Err(err error) *zerolog.Event {
	return l.update().Err(err)
}

func (l *Logger) Write(p []byte) (n int, err error) {
	return l.update().Write(p)
}

func (l *Logger) WithLevel(level zerolog.Level) *zerolog.Event {
	return l.update().WithLevel(level)
}

func (l *Logger) With(with func(zerolog.Context) zerolog.Context) *Logger {
	if with == nil {
		return l
	}
	// TODO: this is inefficient as it will construct multiple contexts and
	// loggers, by daisy chaining through any stacked layers of builders.
	return newFrom(func() (int32, zerolog.Logger) {
		c, ll := l.b()
		ll = with(ll.With()).Logger()
		return c, ll
	})
}

func (l *Logger) Level(level zerolog.Level) *Logger {
	return newFrom(func() (int32, zerolog.Logger) {
		c, ll := l.b()
		ll = ll.Level(level)
		return c, ll
	})
}
