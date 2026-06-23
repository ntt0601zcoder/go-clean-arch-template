package ginx

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthChecker probes one dependency for readiness.
type HealthChecker func(ctx context.Context) error

// LivenessHandler reports the process is up. Used by GET /liveness and /health.
func LivenessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

// ReadinessHandler runs all checks and returns 200 only if every dependency is
// healthy, else 503 with per-check detail.
func ReadinessHandler(checks map[string]HealthChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		ready := true
		details := make(map[string]string, len(checks))
		for name, check := range checks {
			if err := check(ctx); err != nil {
				ready = false
				details[name] = err.Error()
			} else {
				details[name] = "ok"
			}
		}

		status := http.StatusOK
		if !ready {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"ready": ready, "details": details})
	}
}
