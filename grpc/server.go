package grpc

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"go.6river.tech/gosix/ent"
	"go.6river.tech/gosix/logging"
	"go.6river.tech/gosix/registry"
	"go.6river.tech/gosix/server"
)

type GrpcInitializer func(context.Context, *grpc.Server, *registry.Registry, ent.EntClient) error

type grpcServer struct {
	defaultPort  int
	offset       int
	realPort     int
	listenConfig *net.ListenConfig
	opts         []grpc.ServerOption

	initializers []GrpcInitializer

	logger *logging.Logger
	server *grpc.Server
}

func (s *grpcServer) Name() string {
	if s.realPort != 0 && s.realPort != s.defaultPort+s.offset {
		return fmt.Sprintf("grpc@%d+%d=%d", s.defaultPort, s.offset, s.realPort)
	} else {
		return fmt.Sprintf("grpc@%d", s.defaultPort+s.offset)
	}
}

func (s *grpcServer) Initialize(ctx context.Context, services *registry.Registry, client ent.EntClient) error {
	s.realPort = server.ResolvePort(s.defaultPort, s.offset)
	s.logger = logging.GetLogger("server/grpc/" + strconv.Itoa(s.realPort))

	if s.opts == nil {
		// TODO: add logging & prometheus interceptors
		s.opts = []grpc.ServerOption{
			grpc.ChainUnaryInterceptor(
				s.logUnary,
				grpc_prometheus.UnaryServerInterceptor,
			),
			grpc.ChainStreamInterceptor(
				s.logStream,
				grpc_prometheus.StreamServerInterceptor,
			),
		}
		grpc_prometheus.EnableHandlingTimeHistogram()
	}
	if s.listenConfig == nil {
		s.listenConfig = &net.ListenConfig{}
	}

	s.server = grpc.NewServer(s.opts...)

	for _, i := range s.initializers {
		if err := i(ctx, s.server, services, client); err != nil {
			return err
		}
	}

	// this allows the (C++ based) `grpc_cli` tool to poke at us
	reflection.Register(s.server)

	return nil
}

func (s *grpcServer) logUnary(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	elapsed := time.Since(start)
	var evt *zerolog.Event
	if err == nil {
		evt = s.logger.Trace()
	} else if stat, _ := status.FromError(err); stat.Code() == codes.Canceled || stat.Code() == codes.DeadlineExceeded {
		// cancellation & deadline exceeded errors are boring, keep them at trace
		evt = s.logger.Trace()
	} else {
		evt = s.logger.Error().Err(err)
	}
	evt.
		Str("method", info.FullMethod).
		Dur("latency", elapsed).
		Msg("unary")
	return resp, err
}

func (s *grpcServer) logStream(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	start := time.Now()
	cs := &countingStream{ServerStream: ss}
	err := handler(srv, cs)
	elapsed := time.Since(start)
	var evt *zerolog.Event
	if err == nil {
		evt = s.logger.Trace()
	} else {
		evt = s.logger.Error().Err(err)
	}
	evt.
		Str("method", info.FullMethod).
		Dur("latency", elapsed).
		Uint64("sent", cs.numSent).
		Uint64("received", cs.numReceived).
		Uint64("errd", cs.numError).
		Msg("stream")
	return err
}

type countingStream struct {
	grpc.ServerStream
	numSent, numReceived, numError uint64
}

func (cs *countingStream) SendMsg(m interface{}) error {
	err := cs.ServerStream.SendMsg(m)
	if err != nil {
		atomic.AddUint64(&cs.numError, 1)
		return err
	}
	atomic.AddUint64(&cs.numSent, 1)
	return nil
}
func (cs *countingStream) RecvMsg(m interface{}) error {
	err := cs.ServerStream.RecvMsg(m)
	if err != nil {
		atomic.AddUint64(&cs.numError, 1)
		return err
	}
	atomic.AddUint64(&cs.numReceived, 1)
	return nil
}

func (s *grpcServer) Start(ctx context.Context, ready chan<- struct{}) error {
	defer func() {
		if ready != nil {
			close(ready)
		}
	}()

	s.logger.Trace().Msg("startup requested")

	// run the listener on the background context, we'll separately monitor ctx to
	// stop the server
	l, err := s.listenConfig.Listen(context.Background(), "tcp", fmt.Sprintf(":%d", s.realPort))
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to open gRPC listen socket")

		return errors.Wrap(err, "Failed to open listen port for grpc server")
	}

	// ask the server to stop when the context is canceled
	go func() {
		<-ctx.Done()
		ss := s.server
		if ss != nil {
			s.logger.Trace().Msg("Stopping gRPC server")
			s.server.GracefulStop()
		}
		// else someone beat us to it
	}()

	s.logger.Info().Int("port", s.realPort).Msg("gRPC server listening")
	close(ready)
	ready = nil
	err = s.server.Serve(l)
	if errors.Is(err, grpc.ErrServerStopped) {
		// this isn't an error
		err = nil
	}
	if err != nil {
		s.logger.Error().Err(err).Msg("Service failed")
	} else {
		s.logger.Trace().Msg("Service stopped")
	}
	return err
}

func (s *grpcServer) Cleanup(context.Context, *registry.Registry) error {
	if s.server != nil {
		// just in case, do we really need this?
		s.server.Stop()
		s.server = nil
	}

	return nil
}

func NewGrpcService(
	port, offset int,
	opts []grpc.ServerOption,
	initializers ...GrpcInitializer,
) *grpcServer {
	return &grpcServer{
		defaultPort:  port,
		offset:       offset,
		opts:         opts,
		initializers: initializers,
	}
}
