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

	"go.6river.tech/gosix/ent"
)

type Service interface {
	// Name describes the specific service for use in logging and status reports
	Name() string

	// Initialize should do any prep work for the service, but not actually start
	// it yet. The context should only be used for the duration of the initialization.
	Initialize(context.Context, *Registry, ent.EntClientBase) error

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
