// Package ginx holds Gin middleware and ops handlers (health, metrics, pprof,
// swagger) shared by the HTTP transports.
package ginx

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/correlation"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/httpx"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/metrics"
)

// opsPaths are excluded from request logging/metrics to avoid noise & cardinality.
var opsPaths = map[string]bool{
	"/liveness": true, "/readiness": true, "/health": true, "/metrics": true,
}

func isOps(path string) bool {
	if opsPaths[path] {
		return true
	}
	// /debug/pprof/* and /swagger/* prefixes
	return len(path) >= 7 && (path[:7] == "/debug/" || (len(path) >= 9 && path[:9] == "/swagger/"))
}

// CorrelationMiddleware ensures every request carries a correlation id (from the
// inbound header or freshly generated) in its context and echoes it back.
func CorrelationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(correlation.HeaderKey)
		ctx := c.Request.Context()
		if id != "" {
			ctx = correlation.With(ctx, id)
		} else {
			ctx, id = correlation.Ensure(ctx)
		}
		c.Request = c.Request.WithContext(ctx)
		c.Writer.Header().Set(correlation.HeaderKey, id)
		c.Next()
	}
}

// LoggerMiddleware logs one structured line per (non-ops) request.
func LoggerMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isOps(c.FullPath()) {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		log.LogAttrs(c.Request.Context(), slog.LevelInfo, "http request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.FullPath()),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", time.Since(start)),
			slog.String("correlation_id", correlation.Get(c.Request.Context())),
		)
	}
}

// MetricsMiddleware records request counts/latency by method and route template.
func MetricsMiddleware(m *metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		if path == "" || isOps(path) {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		m.ObserveHTTP(c.Request.Method, path, c.Writer.Status(), time.Since(start))
	}
}

// TracingMiddleware wires OpenTelemetry spans for each request.
func TracingMiddleware(serviceName string) gin.HandlerFunc {
	return otelgin.Middleware(serviceName)
}

// ErrorMiddleware renders the last handler error (if any) through httpx, so
// handlers can simply `c.Error(err); return` and stay free of transport codes.
func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}
		he := httpx.NewError(c.Errors.Last().Err)
		c.JSON(he.Status, he.Body)
	}
}

// Recovery converts panics into a 500 JSON error using slog.
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, err any) {
		log.ErrorContext(c.Request.Context(), "panic recovered", slog.Any("panic", err))
		he := httpx.NewError(nil)
		c.AbortWithStatusJSON(he.Status, he.Body)
	})
}
