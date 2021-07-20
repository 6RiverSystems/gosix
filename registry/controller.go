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
	"net/http"

	"github.com/gin-gonic/gin"
)

// Controller defines the interface for a controller that can register its
// routes against a (gin) router.
type Controller interface {
	Register(*Registry, gin.IRouter) error
}

// StartupController defines a controller that receives a notification for app
// Startup, to do any extra initialization it needs. Calls to Startup for each
// registered Controller may be run concurrently on multiple goroutines. Note
// that controllers MUST NOT register new routes or middleware here, they MUST
// do that in their Register function.
type StartupController interface {
	Controller
	Startup(context.Context, gin.IRouter, *http.Server) error
}

// ShutdownController defines a controller that receives a notification for app
// shutdown, to do any cleanup it needs. Calls to Shutdown for each registered
// Controller may be run concurrently on multiple goroutines.
type ShutdownController interface {
	Controller
	Shutdown(context.Context, gin.IRouter, *http.Server) error
}

// LifecycleController is a shorthand to represent a controller that implements
// both Startup and Shutdown methods.
type LifecycleController interface {
	Controller
	StartupController
	ShutdownController
}
