package registry

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"

	"go.6river.tech/gosix/ent"
	"go.6river.tech/gosix/logging"
)

type Registry struct {
	svcMu       sync.Mutex
	allServices []Service
	allReadies  []chan struct{}

	ctlMu          sync.Mutex
	allControllers []Controller

	initialized   bool
	started       bool
	cancelRunning func()
	runningGroup  *errgroup.Group
	runningCtx    context.Context

	_logger     *logging.Logger
	_loggerOnce sync.Once
}

var ServicesKey = PointerAt("services", (*Registry)(nil))

func Services(vs Values) (*Registry, bool) {
	if s, ok := vs.Value(ServicesKey); ok {
		return s.(*Registry), true
	}
	return nil, false
}

func (r *Registry) RunDefault(ctx context.Context, client ent.EntClient, logger *logging.Logger) error {
	clean := false
	var err error
	defer func() {
		if clean && err == nil {
			logger.Info().Msg("app exiting cleanly")
		} else {
			logger.Warn().Msg("app shutdown after error")
		}
	}()

	err = r.InitializeServices(ctx, client)
	defer func() {
		if !clean {
			r.RequestStopServices()
			r.WaitServices() // nolint:errcheck // don't care, we know it failed
		}
		scErr := r.CleanupServices(ctx)
		if scErr != nil {
			// this will probably be a panic-within-a-panic if it happens
			logger.Error().Err(scErr).Msg("Failed to cleanup services")
			if err == nil {
				err = scErr
			}
		}
	}()
	if err != nil {
		return err
	}

	if err = r.StartServices(ctx); err != nil {
		return err
	}
	if err = r.WaitAllReady(ctx); err != nil {
		return err
	}
	logger.Info().Msgf("All %d services ready", len(r.allReadies))

	if err = r.WaitServices(); err != nil {
		logger.Warn().Err(err).Msg("app exiting after error")
	}

	clean = true
	return err
}
