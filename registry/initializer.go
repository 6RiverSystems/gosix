package registry

import (
	"context"

	"go.6river.tech/gosix/ent"
)

type initializingService struct {
	name        string
	initializer func(ctx context.Context, services *Registry, client ent.EntClient) error
	starter     func(ctx context.Context) error
}

// NewInitializer creates a new service that just does simple initialization
// steps at app startup, instead of actually running a background service
func NewInitializer(
	name string,
	initializer func(ctx context.Context, services *Registry, client ent.EntClient) error,
	starter func(ctx context.Context) error,
) *initializingService {
	return &initializingService{name, initializer, starter}
}

func (s *initializingService) Name() string {
	return s.name
}

// Initialize should do any prep work for the service, but not actually start
// it yet. The context should only be used for the duration of the initialization.
func (s *initializingService) Initialize(ctx context.Context, services *Registry, client ent.EntClient) error {
	if s.initializer != nil {
		return s.initializer(ctx, services, client)
	}
	return nil
}

func (s *initializingService) Start(ctx context.Context, ready chan<- struct{}) error {
	defer close(ready)
	if s.starter != nil {
		return s.starter(ctx)
	}
	return nil
}

func (s *initializingService) Cleanup(context.Context, *Registry) error {
	return nil
}
