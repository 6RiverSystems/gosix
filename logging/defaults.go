package logging

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

var defaultLoggingOnce = &sync.Once{}

func ConfigureDefaultLogging() {
	defaultLoggingOnce.Do(func() {
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
		// always log in UTC, with accurate timestamps
		zerolog.TimestampFunc = func() time.Time {
			return time.Now().UTC()
		}
		zerolog.TimeFieldFormat = time.RFC3339Nano
		// NodeJS/bunyan uses "msg" for MessageFieldName, but that's bad for LogDNA,
		// so don't do that here; do make error logging consistent with NodeJS however
		zerolog.ErrorFieldName = "err"
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

		levelStr := os.Getenv("LOG_LEVEL")
		if levelStr != "" {
			levelStr = strings.ToLower(levelStr)
			level, err := zerolog.ParseLevel(levelStr)
			if err != nil {
				panic(err)
			}
			zerolog.SetGlobalLevel(level)
		} else {
			// default to info logging in production, else debug logging
			if os.Getenv("NODE_ENV") == "production" {
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
			} else {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			}
		}

		log.Logger = zerolog.New(os.Stdout).
			With().
			Timestamp().
			Logger()

		log.Info().Msg("Logging initialized")
	})
}
