package logging

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// logging registry is here instead of in the registry package to avoid circular
// and unnecessary dependencies

var configGeneration int32
var configMutex sync.Mutex
var componentLevel = map[string]zerolog.Level{}

func configLevel(component string) (generation int32, level zerolog.Level) {
	configMutex.Lock()
	defer configMutex.Unlock()
	c := atomic.LoadInt32(&configGeneration)
	l, ok := componentLevel[component]
	for !ok {
		lastSlash := strings.LastIndexByte(component, '/')
		if lastSlash < 1 {
			l = zerolog.GlobalLevel()
			break
		}
		component = component[0:lastSlash]
		l, ok = componentLevel[component]
	}
	return c, l
}

func contextBuilder(component string, with func(zerolog.Context) zerolog.Context) logBuilder {
	return func() (int32, zerolog.Logger) {
		c, level := configLevel(component)
		ctx := log.Logger.With()
		if component != "" {
			ctx = ctx.Str("component", component)
		}
		if with != nil {
			ctx = with(ctx)
		}
		return c, ctx.Logger().Level(level)
	}
}

// GetLogger creates a logger for the given component name. Hierarchies in
// components should be represented with `/` characters in their name.
func GetLogger(component string) *Logger {
	return newFrom(contextBuilder(component, nil))
}

// GetLoggerWith creates a logger for the given component name and custom
// context configuration function. Hierarchies in components should be
// represented with `/` characters in their name.
func GetLoggerWith(component string, with func(zerolog.Context) zerolog.Context) *Logger {
	return newFrom(contextBuilder(component, with))
}

// SetComponentLevel changes the log level for a given component. If children is
// true, it will also change the log level for any child components that have
// been customized (by deleting that customization so that they inherit the log
// level from their parent).
func SetComponentLevel(component string, children bool, level zerolog.Level) {
	configMutex.Lock()
	defer configMutex.Unlock()
	if component == "" {
		// this is a weird special case
		zerolog.SetGlobalLevel(level)
		if children {
			componentLevel = map[string]zerolog.Level{}
		}
	} else {
		// TODO: what should this do for child components?
		componentLevel[component] = level
		p := component + "/"
		for c := range componentLevel {
			if strings.HasPrefix(c, p) {
				delete(componentLevel, c)
			}
		}
	}
	atomic.AddInt32(&configGeneration, 1)
}

// ComponentLevels returns a _copy_ of the currently configured component level map
func ComponentLevels() map[string]zerolog.Level {
	configMutex.Lock()
	defer configMutex.Unlock()
	ret := make(map[string]zerolog.Level, len(componentLevel)+1)
	ret[""] = zerolog.GlobalLevel()
	for k, v := range componentLevel {
		ret[k] = v
	}
	return ret
}

// LeveledLogger implements the interface of the same name from
// github.com/hashicorp/go-retryablehttp
type LeveledLogger struct {
	l *Logger
}

// Leveled returns a LeveledLogger wrapper for the given Logger
func Leveled(l *Logger) LeveledLogger {
	return LeveledLogger{l}
}

func logPairs(event *zerolog.Event, msg string, keysAndValues ...interface{}) {
	for i := 0; i < len(keysAndValues); i += 2 {
		key := keysAndValues[i].(string)
		if err, ok := keysAndValues[i+1].(error); ok {
			if key == zerolog.ErrorFieldName || key == "error" {
				// only .Err() obeys stack printing
				event = event.Err(err)
			} else {
				event = event.AnErr(key, err)
			}
		} else {
			event = event.Interface(key, keysAndValues[i+1])
		}
	}
	event.Msg(msg)
}

func (l LeveledLogger) Error(msg string, keysAndValues ...interface{}) {
	logPairs(l.l.Error(), msg, keysAndValues...)
}
func (l LeveledLogger) Info(msg string, keysAndValues ...interface{}) {
	logPairs(l.l.Info(), msg, keysAndValues...)
}
func (l LeveledLogger) Debug(msg string, keysAndValues ...interface{}) {
	logPairs(l.l.Debug(), msg, keysAndValues...)
}
func (l LeveledLogger) Warn(msg string, keysAndValues ...interface{}) {
	logPairs(l.l.Warn(), msg, keysAndValues...)
}
