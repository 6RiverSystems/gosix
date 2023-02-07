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
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"

	"go.6river.tech/gosix/ent"
	"go.6river.tech/gosix/faults"
	"go.6river.tech/gosix/logging"
)

type Registry struct {
	svcMu       sync.Mutex
	allServices []Service
	allReadies  []chan struct{}
	faults      *faults.Set
	MutableValues

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

func New(appName string, parent Values) *Registry {
	ret := &Registry{
		faults:        faults.NewSet(appName),
		MutableValues: ChildValues(parent, "registry"),
	}
	ret.faults.MustRegister(prometheus.DefaultRegisterer)
	return ret
}

var RegistryKey = PointerAt[Registry]("registry")

func GetRegistry(vs Values) (*Registry, bool) {
	if s, ok := vs.Value(RegistryKey); ok {
		return s.(*Registry), true
	}
	return nil, false
}

func (r *Registry) Faults() *faults.Set {
	return r.faults
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
