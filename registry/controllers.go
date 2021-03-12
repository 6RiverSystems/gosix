package registry

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

// AddController appends a controller instance to the list of those to register
// against a (gin) Router (typically an Engine)
func (r *Registry) AddController(c Controller) {
	r.ctlMu.Lock()
	defer r.ctlMu.Unlock()

	if r.cancelRunning != nil {
		// TODO: actually we could support this if we needed to
		panic(errors.New("Cannot add controllers after services have been started"))
	}

	r.allControllers = append(r.allControllers, c)
}

// RegisterControllers notifies every added Controller to RegisterControllers
// itself with the given Router.
func (r *Registry) RegisterControllers(router gin.IRouter) error {
	for _, c := range r.allControllers {
		if err := c.Register(r, router); err != nil {
			return err
		}
	}
	return nil
}

// StartControllers notifies all added StartupControllers that the given Router
// (typically an Engine) and Server combo are being shut down.
func (r *Registry) StartControllers(ctx context.Context, router gin.IRouter, s *http.Server) error {
	wg := &errgroup.Group{}
	for _, c := range r.allControllers {
		if sc, ok := c.(StartupController); ok {
			wg.Go(func() error { return sc.Startup(ctx, router, s) })
		}
	}
	return wg.Wait()
}

// ShutdownControllers notifies all added ShutdownControllers that the given Router
// (typically an Engine) and Server combo are being shut down.
func (r *Registry) ShutdownControllers(ctx context.Context, router gin.IRouter, s *http.Server) error {
	wg := &errgroup.Group{}
	for _, c := range r.allControllers {
		if sc, ok := c.(ShutdownController); ok {
			wg.Go(func() error { return sc.Shutdown(ctx, router, s) })
		}
	}
	return wg.Wait()
}

// HandlerMap defines a mapping from pairs of (method, path) to a handler
// function for a route. It is used in RegisterMap for a controller to add its
// routes in a table-driven format.
type HandlerMap = map[struct{ Method, Path string }]gin.HandlerFunc

// MethodAny is a special value to pass for the Method key in a HandlerMap to
// call RouterGroup.Any instead of RouterGroup.METHOD. It is chosen so that it
// cannot overlap with any valid HTTP method.
const MethodAny = "any"

// RegisterMap is a helper for making many calls to RouterGroup.METHOD(...),
// using a table-driven approach. As a special case, if the Method in a map
// entry is "any", it will call RouterGroup.Any instead of a method-specific
// handler. As another special case, `nil` handlers will be replaced with a
// default handler that returns HTTP 501 Not Implemented.
func (r *Registry) RegisterMap(router gin.IRouter, root string, endpoints HandlerMap) *gin.RouterGroup {
	rg := router.Group(root)

	for route, handler := range endpoints {
		if handler == nil {
			handler = notImplemented
		}
		if route.Method == MethodAny {
			rg.Any(route.Path, handler)
		} else {
			rg.Handle(route.Method, route.Path, handler)
		}
	}

	return rg
}

func notImplemented(c *gin.Context) {
	c.String(http.StatusNotImplemented, "%s", http.StatusText(http.StatusNotImplemented))
	// don't Abort, this is just the defined response
}
