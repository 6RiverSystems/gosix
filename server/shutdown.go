package server

import (
	"context"
	"net/http"
	"os"
	"time"

	"go.6river.tech/gosix/logging"
	"go.6river.tech/gosix/registry"
)

func Shutdown(s *http.Server, services *registry.Registry, graceful bool) {
	logger := logging.GetLogger("shutdown")
	timeout := time.Second
	if graceful {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	logger.Info().Msg("Starting server shutdown")
	if err := s.Shutdown(ctx); err != nil {
		logger.Err(err).Msg("Graceful shutdown did not complete")
	} else {
		logger.Info().Msg("Graceful shutdown ok")
	}

	services.RequestStopServices()

	// if the main goroutine exits, the app exits, and so it doesn't matter if
	// this code runs
	if !graceful {
		go func() {
			// if main goroutine doesn't exit within one more second, just die
			time.Sleep(time.Second)
			logger.Info().Msg("Forcing exit after timeout")
			os.Exit(1)
		}()
	}
}
