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
