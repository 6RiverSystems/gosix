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
	"github.com/pkg/errors"
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

func (s *httpService) Initialize(ctx context.Context, reg *registry.Registry, _ ent.EntClient) error {
	s.realPort = ResolvePort(s.defaultPort, s.offset)
	s.logger = logging.GetLogger("server/gin-http/" + strconv.Itoa(s.realPort))
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.realPort),
		Handler: s.handler,
		// BaseContext will be set during Start
	}

	if err := reg.StartControllers(ctx, s.handler, s.server); err != nil {
		return errors.Wrap(err, "Registry startup failed")
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
		return errors.Wrap(err, "Registry shutdown failed")
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
