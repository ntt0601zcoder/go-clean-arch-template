// Package httpserver wraps a Gin engine + net/http.Server with Start/Stop hooks
// suitable for FX lifecycle (Start is non-blocking; Stop drains gracefully).
package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Server is an HTTP server built around a mutable *gin.Engine. Register routes
// on Engine() before Start; the underlying http.Server serves the same engine.
type Server struct {
	engine *gin.Engine
	srv    *http.Server
	addr   string
	log    *slog.Logger
}

// New builds a server listening on addr with the given middleware applied.
func New(addr string, log *slog.Logger, middlewares ...gin.HandlerFunc) *Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(middlewares...)
	return &Server{
		engine: engine,
		addr:   addr,
		log:    log.With(slog.String("server", "http"), slog.String("addr", addr)),
		srv: &http.Server{
			Addr:              addr,
			Handler:           engine,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

// Engine returns the Gin engine for route registration.
func (s *Server) Engine() *gin.Engine { return s.engine }

// Start begins serving in a background goroutine and returns immediately.
func (s *Server) Start(_ context.Context) error {
	go func() {
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("http server stopped with error", slog.Any("err", err))
		}
	}()
	s.log.Info("http server started")
	return nil
}

// Stop gracefully drains in-flight requests.
func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("http server stopping")
	return s.srv.Shutdown(ctx)
}
