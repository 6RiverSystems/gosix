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
