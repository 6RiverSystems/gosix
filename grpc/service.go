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

package grpc

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/grpc"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	entcommon "go.6river.tech/gosix/ent"
	"go.6river.tech/gosix/logging"
	"go.6river.tech/gosix/registry"
	"go.6river.tech/gosix/server"
)

type gatewayService struct {
	name           string
	port, offset   int
	routes         gin.IRoutes
	paths          []string
	grpcServiceTag registry.ServiceTag
	initHandlers   func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error

	logger   *logging.Logger
	endpoint string
	conn     *grpc.ClientConn
	services *registry.Registry
}

func NewGatewayService(
	name string,
	port, offset int,
	routes gin.IRoutes,
	paths []string,
	grpcServiceTag registry.ServiceTag,
	initHandlers func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error,
) *gatewayService {
	return &gatewayService{
		name: name,
		port: port, offset: offset,
		routes:       routes,
		paths:        paths,
		initHandlers: initHandlers,
	}
}

func (s *gatewayService) Name() string {
	return "grpc-http-gateway(" + s.name + ")"
}

func (s *gatewayService) Initialize(_ context.Context, services *registry.Registry, _ entcommon.EntClient) error {
	s.logger = logging.GetLogger("grpc/http-gateway")
	s.endpoint = "localhost:" + strconv.Itoa(server.ResolvePort(s.port, s.offset))
	s.services = services
	return nil
}

func (s *gatewayService) Start(ctx context.Context, ready chan<- struct{}) error {
	defer close(ready)

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		// retry if we get connection refused, as this proxy might start before
		// the grpc server starts ... this doesn't really seem to work however
		grpc.FailOnNonTempDialError(false),
	}

	<-s.services.ReadyWaiter(s.grpcServiceTag)

	var err error
	if s.conn, err = grpc.Dial(s.endpoint, opts...); err != nil {
		return err
	}
	if err = s.initHandlers(ctx, mux, s.conn); err != nil {
		return err
	}
	for _, p := range s.paths {
		s.routes.Any(p, gin.WrapF(func(w http.ResponseWriter, r *http.Request) {
			for {
				if s.services.ServicesStarted() {
					break
				}
				select {
				case <-r.Context().Done():
					// abort
					return
				case <-time.After(time.Millisecond):
					// re-check
				}
			}
			mux.ServeHTTP(w, r)
		}))
	}

	// service doesn't need to do anything, just is used for delayed registration
	return nil
}

func (s *gatewayService) Cleanup(context.Context, *registry.Registry) error {
	if err := s.conn.Close(); err != nil {
		s.logger.Error().Err(err).Msgf("Failed to close conn to %s", s.endpoint)
		return err
	}
	return nil
}
