// Copyright (c) 2021 6 River Systems
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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
