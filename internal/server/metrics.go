package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hrodrig/gghstats/internal/version"
)

// MetricsPath is the Prometheus scrape endpoint (GET only).
const MetricsPath = "/metrics"

func newMetricsRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	build := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gghstats_build_info",
			Help: "Build metadata (value is always 1).",
		},
		[]string{"version", "commit"},
	)
	build.WithLabelValues(version.Version, version.Commit).Set(1)
	reg.MustRegister(build)
	return reg
}

func metricsExporter(reg prometheus.Gatherer) http.Handler {
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// wrapWithHTTPMetrics records request counts and latencies with a low-cardinality route label.
func wrapWithHTTPMetrics(reg prometheus.Registerer, h http.Handler) http.Handler {
	req := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gghstats_http_requests_total",
			Help: "HTTP requests processed, by method, normalized route, and status code.",
		},
		[]string{"method", "route", "status"},
	)
	dur := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gghstats_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "route"},
	)
	reg.MustRegister(req, dur)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := metricsRouteLabel(r)
		sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		h.ServeHTTP(sr, r)
		status := strconv.Itoa(sr.status)
		req.WithLabelValues(r.Method, route, status).Inc()
		dur.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func metricsRouteLabel(r *http.Request) string {
	p := r.URL.Path
	switch {
	case p == MetricsPath:
		return "metrics"
	case p == HealthzPath:
		return "health"
	case strings.HasPrefix(p, "/static/"):
		return "static"
	case p == "/api/repos":
		return "api_repos"
	case p == "/":
		return "index"
	}
	parts := strings.Split(strings.Trim(p, "/"), "/")
	if len(parts) == 2 && parts[0] != "" && parts[0] != "static" && parts[0] != "api" {
		return "repo"
	}
	return "other"
}
