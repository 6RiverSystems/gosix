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
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"go.6river.tech/gosix/ent"
	"go.6river.tech/gosix/logging"
)

type ServiceTag int

// AddService registers a Service instance to be affected by future calls to
// InitializeAll, StartAll, StopAll, and CleanupAll. It is an error (panic) to
// call AddService when StartAll is running.
func (r *Registry) AddService(s Service) ServiceTag {
	r.svcMu.Lock()
	defer r.svcMu.Unlock()

	if r.cancelRunning != nil {
		panic(errors.New("Cannot add services after they have been started"))
	}
	tag := len(r.allServices)
	r.allServices = append(r.allServices, s)
	r.allReadies = append(r.allReadies, make(chan struct{}))
	return ServiceTag(tag)
}

// we don't create a logger as a package-level variable, because it would get
// initialized before logging is configured
func (r *Registry) logger() *logging.Logger {
	r._loggerOnce.Do(func() {
		// TODO: registry names
		r._logger = logging.GetLogger("services/registry")
	})
	return r._logger
}

// InitializeServices calls the Initialize method on all registered services, in
// parallel in goroutines.
func (r *Registry) InitializeServices(ctx context.Context, client ent.EntClient) error {
	if r.initialized {
		return errors.New("Cannot re-initialize services without cleanup")
	}

	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eg, egCtx := errgroup.WithContext(cancelCtx)

	r.logger().Info().Msgf("Initializing %d services", len(r.allServices))

	for _, s := range r.allServices {
		s := s
		eg.Go(func() error {
			return s.Initialize(egCtx, r, client)
		})
	}

	if err := eg.Wait(); err != nil {
		err2 := r.CleanupServices(ctx)
		// TODO: error during error recovery, yuck
		r.logger().Error().Err(err2).Msg("Failed to cleanup after failure to initialize")
		return err
	}
	r.initialized = true
	return nil
}

func (r *Registry) ServicesInitialized() bool {
	return r.initialized
}

// StartServices starts all registered services in individual goroutines. It is an
// error to call StartServices if it is already running. StartServices returns once all
// services are started, it does not wait for them to complete. Use StopAll to
// request they stop and wait for the result.
func (r *Registry) StartServices(ctx context.Context) error {
	if !r.initialized {
		return errors.New("Cannot start without initializing first")
	}
	if r.ServicesStarted() || r.cancelRunning != nil {
		return errors.New("Cannot start again when services are already running")
	}

	var cancelCtx context.Context
	cancelCtx, r.cancelRunning = context.WithCancel(ctx)
	r.runningGroup, r.runningCtx = errgroup.WithContext(cancelCtx)

	r.logger().Info().Msgf("Starting %d services", len(r.allServices))

	for i, s := range r.allServices {
		s := s
		ready := r.allReadies[i]
		r.runningGroup.Go(func() error {
			return s.Start(r.runningCtx, ready)
		})
	}

	r.svcMu.Lock()
	defer r.svcMu.Unlock()
	r.started = true

	return nil
}

func (r *Registry) ServicesStarted() bool {
	r.svcMu.Lock()
	defer r.svcMu.Unlock()
	return r.started
}

func (r *Registry) ReadyWaiter(tag ServiceTag) <-chan struct{} {
	r.svcMu.Lock()
	defer r.svcMu.Unlock()
	return r.allReadies[tag]
}

func (r *Registry) WaitAllReady(ctx context.Context) error {
	if !r.ServicesStarted() {
		panic(errors.New("Cannot wait for services to be ready until they have been started"))
	}
	for i := range r.allReadies {
		if err := r.waitReady(ctx, ServiceTag(i)); err != nil {
			return err
		}
	}
	return nil
}

var ServiceNotFoundError = fmt.Errorf("service not found")

func (r *Registry) WaitReadyByName(ctx context.Context, name string) error {
	if !r.ServicesStarted() {
		panic(errors.New("Cannot wait for services to be ready until they have been started"))
	}
	found := false
	for i := range r.allReadies {
		if r.allServices[i].Name() != name {
			continue
		}
		found = true
		if err := r.waitReady(ctx, ServiceTag(i)); err != nil {
			return err
		}
	}
	if !found {
		return ServiceNotFoundError
	}
	return nil
}

func (r *Registry) waitReady(ctx context.Context, tag ServiceTag) error {
	s := r.allServices[tag]
	ready := r.allReadies[tag]
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.DeadlineExceeded) {
				r.logger().Error().
					Int("serviceTag", int(tag)).
					Str("service", s.Name()).
					Msg("Timed out waiting for service to be ready")
			}
			return err
		case <-r.runningCtx.Done():
			err := r.runningGroup.Wait()
			if err != nil {
				r.logger().Error().
					Err(err).
					Int("serviceTag", int(tag)).
					Str("service", s.Name()).
					Msg("Services failed while waiting for service to be ready")
			}
			return err
		case <-time.After(time.Second):
			r.logger().Warn().
				Int("serviceTag", int(tag)).
				Str("service", s.Name()).
				Msg("Service is slow to get ready")
		case <-ready:
			return nil
		}
	}
}

// WaitServices will wait for the running services, if any, to all end. It will
// return the resulting error, if any.
func (r *Registry) WaitServices() error {
	if r.runningGroup == nil {
		return nil
	}
	err := r.runningGroup.Wait()
	if err == context.Canceled || err == context.DeadlineExceeded {
		// this just means yes, we stopped it
		err = nil
	}
	// flag that services are no longer running
	r.svcMu.Lock()
	defer r.svcMu.Unlock()
	r.started = false
	return err
}

// RequestStopServices requests all running services stop, by cancelling their
// Context objects
func (r *Registry) RequestStopServices() {
	r.svcMu.Lock()
	defer r.svcMu.Unlock()

	if !r.initialized {
		return
	}
	if r.cancelRunning != nil {
		r.logger().Info().Msgf("Stopping %d services", len(r.allServices))
		r.cancelRunning()
	}
}

func (r *Registry) CleanupServices(ctx context.Context) error {
	if !r.initialized {
		return nil
	}
	if r.ServicesStarted() {
		return errors.New("Cannot cleanup when services are still running")
	}

	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eg, egCtx := errgroup.WithContext(cancelCtx)

	r.logger().Info().Msgf("Cleaning up %d services", len(r.allServices))

	for _, s := range r.allServices {
		s := s
		eg.Go(func() error {
			return s.Cleanup(egCtx, r)
		})
	}

	err := eg.Wait()
	if err == context.Canceled || err == context.DeadlineExceeded {
		// this just means yes, we stopped it
		err = nil
	}

	r.svcMu.Lock()
	defer r.svcMu.Unlock()
	r.cancelRunning = nil
	r.initialized = false
	return err
}
