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

package registry

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.6river.tech/gosix/ent"
	"go.6river.tech/gosix/logging"
)

type signalService struct {
	signals  []os.Signal
	logger   *logging.Logger
	notify   chan os.Signal
	services *Registry
}

func (s *signalService) Name() string {
	return fmt.Sprintf("shutdown-on-signals(%v)", s.signals)
}

func (s *signalService) Initialize(_ context.Context, services *Registry, _ ent.EntClient) error {
	s.logger = logging.GetLogger("signals")
	s.notify = make(chan os.Signal, 5)
	s.services = services
	return nil
}

func (s *signalService) Start(ctx context.Context, ready chan<- struct{}) error {
	signal.Notify(s.notify, s.signals...)
	close(ready)
	for {
		select {
		case signal, ok := <-s.notify:
			if !ok {
				return nil
			}
			s.logger.Info().Stringer("signal", signal).Msg("Shutting down on signal")
			s.services.RequestStopServices()
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *signalService) Cleanup(ctx context.Context, _ *Registry) error {
	if s.notify != nil {
		signal.Stop(s.notify)
		close(s.notify)
		s.notify = nil
	}
	return nil
}

func RegisterDefaultSignalListener(s *Registry) {
	s.AddService(&signalService{
		signals: []os.Signal{syscall.SIGINT, syscall.SIGTERM},
	})
}
