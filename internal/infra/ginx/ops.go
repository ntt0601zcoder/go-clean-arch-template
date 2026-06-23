package ginx

import (
	"net/http/pprof"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/metrics"
)

// RegisterHealth mounts /liveness, /readiness and /health.
func RegisterHealth(r gin.IRoutes, checks map[string]HealthChecker) {
	r.GET("/liveness", LivenessHandler())
	r.GET("/health", LivenessHandler())
	r.GET("/readiness", ReadinessHandler(checks))
}

// RegisterMetrics mounts GET /metrics for Prometheus scraping.
func RegisterMetrics(r gin.IRoutes, m *metrics.Metrics) {
	r.GET("/metrics", gin.WrapH(m.Handler()))
}

// RegisterPprof mounts the stdlib net/http/pprof handlers under /debug/pprof.
// Never expose this publicly.
func RegisterPprof(r *gin.Engine) {
	g := r.Group("/debug/pprof")
	g.GET("/", gin.WrapF(pprof.Index))
	g.GET("/cmdline", gin.WrapF(pprof.Cmdline))
	g.GET("/profile", gin.WrapF(pprof.Profile))
	g.GET("/symbol", gin.WrapF(pprof.Symbol))
	g.POST("/symbol", gin.WrapF(pprof.Symbol))
	g.GET("/trace", gin.WrapF(pprof.Trace))
	g.GET("/:profile", gin.WrapF(pprof.Index)) // heap, goroutine, allocs, block, ...
}

// RegisterSwagger mounts the Swagger UI at /swagger/*any. The OpenAPI spec is
// produced by `swag init` into the docs package (blank-imported by the app).
func RegisterSwagger(r *gin.Engine) {
	r.GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))
}
