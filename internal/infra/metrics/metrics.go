// Package metrics owns a dedicated Prometheus registry plus the application
// metric vectors, and exposes the /metrics HTTP handler. A private registry
// (instead of the global default) keeps tests isolated and avoids accidental
// double registration.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds the registry and instruments.
type Metrics struct {
	reg *prometheus.Registry

	httpRequests *prometheus.CounterVec
	httpDuration *prometheus.HistogramVec
	WorkerRuns   *prometheus.CounterVec
}

// New builds the registry, registers Go/process collectors and app metrics.
func New(namespace string) *Metrics {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	m := &Metrics{
		reg: reg,
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: "http", Name: "requests_total",
			Help: "Total HTTP requests by method, path and status.",
		}, []string{"method", "path", "status"}),
		httpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace, Subsystem: "http", Name: "request_duration_seconds",
			Help:    "HTTP request latency by method and path.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),
		WorkerRuns: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: "worker", Name: "runs_total",
			Help: "Total worker job runs by job and result.",
		}, []string{"job", "result"}),
	}
	reg.MustRegister(m.httpRequests, m.httpDuration, m.WorkerRuns)
	return m
}

// Registry exposes the underlying registry (gRPC metrics register onto it too).
func (m *Metrics) Registry() *prometheus.Registry { return m.reg }

// Handler returns the /metrics HTTP handler.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{})
}

// ObserveHTTP records one HTTP request.
func (m *Metrics) ObserveHTTP(method, path string, status int, dur time.Duration) {
	m.httpRequests.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
	m.httpDuration.WithLabelValues(method, path).Observe(dur.Seconds())
}
