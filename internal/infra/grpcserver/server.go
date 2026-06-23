// Package grpcserver wraps a *grpc.Server with the standard health service,
// reflection, configurable interceptors and FX-friendly Start/Stop.
package grpcserver

import (
	"context"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/stats"
)

// Server bundles a grpc.Server with a health.Server.
type Server struct {
	gs     *grpc.Server
	health *health.Server
	addr   string
	log    *slog.Logger
	lis    net.Listener
}

type config struct {
	unary      []grpc.UnaryServerInterceptor
	stats      []stats.Handler
	reflection bool
}

// Option configures the server.
type Option func(*config)

// WithUnaryInterceptors chains unary interceptors (outermost first).
func WithUnaryInterceptors(i ...grpc.UnaryServerInterceptor) Option {
	return func(c *config) { c.unary = append(c.unary, i...) }
}

// WithStatsHandler attaches a stats handler (e.g. otelgrpc for tracing/metrics).
func WithStatsHandler(h stats.Handler) Option {
	return func(c *config) { c.stats = append(c.stats, h) }
}

// WithReflection enables server reflection (grpcurl, evans).
func WithReflection() Option {
	return func(c *config) { c.reflection = true }
}

// New builds the gRPC server. The health service is always registered.
func New(addr string, log *slog.Logger, opts ...Option) *Server {
	var c config
	for _, opt := range opts {
		opt(&c)
	}

	serverOpts := make([]grpc.ServerOption, 0, len(c.stats)+1)
	if len(c.unary) > 0 {
		serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(c.unary...))
	}
	for _, h := range c.stats {
		serverOpts = append(serverOpts, grpc.StatsHandler(h))
	}

	gs := grpc.NewServer(serverOpts...)
	hs := health.NewServer()
	grpc_health_v1.RegisterHealthServer(gs, hs)
	if c.reflection {
		reflection.Register(gs)
	}

	return &Server{
		gs:     gs,
		health: hs,
		addr:   addr,
		log:    log.With(slog.String("server", "grpc"), slog.String("addr", addr)),
	}
}

// GRPC exposes the underlying server so handlers can register their services.
func (s *Server) GRPC() *grpc.Server { return s.gs }

// Health exposes the health server (readiness probes can read its status).
func (s *Server) Health() *health.Server { return s.health }

// Start binds the listener and serves in a background goroutine.
func (s *Server) Start(_ context.Context) error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.lis = lis
	s.health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	go func() {
		if err := s.gs.Serve(lis); err != nil {
			s.log.Error("grpc server stopped with error", slog.Any("err", err))
		}
	}()
	s.log.Info("grpc server started")
	return nil
}

// Stop gracefully stops the server.
func (s *Server) Stop(_ context.Context) error {
	s.log.Info("grpc server stopping")
	s.health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	s.gs.GracefulStop()
	return nil
}
