package registry

import (
	"context"

	"go.6river.tech/gosix/ent"
)

type Service interface {
	// Name describes the specific service for use in logging and status reports
	Name() string

	// Initialize should do any prep work for the service, but not actually start
	// it yet. The context should only be used for the duration of the initialization.
	Initialize(context.Context, *Registry, ent.EntClient) error

	// Start runs the service. It will be invoked on a goroutine, so it should
	// block and not return until the context is canceled, which is how the
	// service is requested to stop. The service must close the ready channel once
	// it is operational, so that any dependent services can know when they are OK
	// to proceed.
	Start(ctx context.Context, ready chan<- struct{}) error

	// Cleanup should release any resources acquired during Initialize. If another
	// service fails during Initialize, Cleanup may be called without Start ever
	// being called. If Start is called, Cleanup will not be called until after it
	// returns.
	Cleanup(context.Context, *Registry) error
}
