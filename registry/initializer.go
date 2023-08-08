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

type initializingService struct {
	name        string
	initializer func(ctx context.Context, services *Registry, client ent.EntClientBase) error
	starter     func(ctx context.Context) error
}

// NewInitializer creates a new service that just does simple initialization
// steps at app startup, instead of actually running a background service
func NewInitializer(
	name string,
	initializer func(ctx context.Context, services *Registry, client ent.EntClientBase) error,
	starter func(ctx context.Context) error,
) *initializingService {
	return &initializingService{name, initializer, starter}
}

func (s *initializingService) Name() string {
	return s.name
}

// Initialize should do any prep work for the service, but not actually start
// it yet. The context should only be used for the duration of the initialization.
func (s *initializingService) Initialize(ctx context.Context, services *Registry, client ent.EntClientBase) error {
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
