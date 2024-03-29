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

package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"

	"go.6river.tech/gosix/ent"
	"go.6river.tech/gosix/logging"
	"go.6river.tech/gosix/registry"
)

type httpService struct {
	defaultPort int
	offset      int
	realPort    int
	handler     *gin.Engine
	logger      *logging.Logger
	server      *http.Server
}

func (s *httpService) Name() string {
	if s.realPort != 0 && s.realPort != s.defaultPort+s.offset {
		return fmt.Sprintf("gin-http(%d+%d=%d)", s.defaultPort, s.offset, s.realPort)
	} else {
		return fmt.Sprintf("gin-http(%d)", s.defaultPort+s.offset)
	}
}

func (s *httpService) Initialize(ctx context.Context, reg *registry.Registry, _ ent.EntClientBase) error {
	s.realPort = ResolvePort(s.defaultPort, s.offset)
	s.logger = logging.GetLogger("server/gin-http/" + strconv.Itoa(s.realPort))
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.realPort),
		Handler: s.handler,
		// BaseContext will be set during Start
	}

	if err := reg.StartControllers(ctx, s.handler, s.server); err != nil {
		return fmt.Errorf("Registry startup failed: %w", err)
	}

	return nil
}

func (s *httpService) Start(ctx context.Context, ready chan<- struct{}) error {
	ctx, _ = registry.WithChildValues(ctx, s.Name())
	s.server.BaseContext = func(l net.Listener) context.Context {
		lctx, _ := registry.WithChildValues(ctx, l.Addr().String())
		return lctx
	}

	// TODO: should listener be started in Initialize?

	// split listen from serve so that we can log when the listen socket is ready
	s.logger.Trace().Str("addr", s.server.Addr).Msgf("listening")
	lc := net.ListenConfig{}
	l, err := lc.Listen(ctx, "tcp", s.server.Addr)
	if err != nil {
		return err
	}
	defer l.Close()
	s.logger.Info().
		// FIXME: get app version via registry.Values injection and report it
		// Str("version", version.SemrelVersion).
		Int("port", s.realPort).
		Msgf("Server is ready")
	eg, egCtx := errgroup.WithContext(ctx)
	// wait for the parent context to be canceled, then request a clean shutdown
	eg.Go(func() error {
		<-egCtx.Done()
		// run shutdown on the background context, since `ctx` is already done
		return s.server.Shutdown(context.Background())
	})
	eg.Go(func() error {
		close(ready)
		err = s.server.Serve(l)
		// ignore graceful shutdowns, they're not a real error
		if err == http.ErrServerClosed {
			err = nil
		}
		return err
	})
	return eg.Wait()
}

func (s *httpService) Cleanup(ctx context.Context, reg *registry.Registry) error {
	if err := s.server.Close(); err != nil {
		s.logger.Err(err).Msg("Server close failed")
		// keep going for the non-graceful cleanup
	}

	// cleanup registered controllers, etc. Note that this assumes the server
	// handler is the gin router/engine, and not some non-gin middleware layer.
	err := reg.ShutdownControllers(ctx, s.handler, s.server)
	if err != nil {
		return fmt.Errorf("Registry shutdown failed: %w", err)
	}

	return err
}

// RegisterHttpService configures an HTTP server in the service registry using
// the given routing engine.
//
// It will listen on `defaultPort` normally, unless overridden by the PORT
// environment variable.
func RegisterHttpService(services *registry.Registry, r *gin.Engine, defaultPort, offset int) registry.ServiceTag {
	return services.AddService(&httpService{
		defaultPort: defaultPort,
		offset:      offset,
		handler:     r,
	})
}

func HttpServer(c *gin.Context) *http.Server {
	return c.Request.Context().Value(http.ServerContextKey).(*http.Server)
}

func Values(c *gin.Context) registry.Values {
	return registry.ContextValues(c.Request.Context())
}

// randomizedPorts is, functionally, a `map[int]int`, and is mostly for testing,
// or other cases where we want to pick a random free TCP port for listening,
// instead of a pre-determined one
var randomizedPorts *sync.Map

func EnableRandomPorts() {
	if randomizedPorts == nil {
		randomizedPorts = new(sync.Map)
	}
}

func getRandomPort(port int) int {
	rPort, ok := randomizedPorts.Load(port)
	if !ok {
		// make & destroy a listener to select a random port, hopefully it will
		// still be available when the real app code needs to use it
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			panic(err)
		}
		defer listener.Close()
		rPort = listener.Addr().(*net.TCPAddr).Port
		var loaded bool
		rPort, loaded = randomizedPorts.LoadOrStore(port, rPort)
		if !loaded {
			logging.GetLogger("server").Info().
				Int("port", port).
				Int("randomPort", rPort.(int)).
				Msg("Using randomized port")
		}
	}
	return rPort.(int)
}

func ResolvePort(defaultBasePort, offset int) int {
	if randomizedPorts != nil {
		return getRandomPort(defaultBasePort + offset)
	} else if port := os.Getenv("PORT"); port != "" {
		listenPort, err := strconv.ParseInt(port, 10, 16)
		if err != nil {
			panic(err)
		}
		return int(listenPort) + offset
	} else {
		return defaultBasePort + offset
	}
}
