package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hrodrig/gghstats/internal/metrics"
	"github.com/hrodrig/gghstats/internal/store"
	"github.com/hrodrig/gghstats/internal/version"
)

// MetricsPath is the Prometheus scrape endpoint (GET only).
const MetricsPath = "/metrics"

// MetricsRegistryConfig configures the Prometheus registry and domain metrics.
type MetricsRegistryConfig struct {
	Store          *store.Store
	DBPath         string
	Filter         string
	PerRepoEnabled bool
}

// NewMetricsRegistry creates a registry with build/runtime collectors and domain metrics.
func NewMetricsRegistry(cfg MetricsRegistryConfig) (*prometheus.Registry, *metrics.Domain) {
	reg := prometheus.NewRegistry()
	registerBuildAndRuntime(reg)
	var dom *metrics.Domain
	if cfg.Store != nil {
		st := cfg.Store
		dom = metrics.RegisterDomain(reg, metrics.DomainConfig{
			Filter:         cfg.Filter,
			DBPath:         cfg.DBPath,
			StoreRepoCount: st.RepoCount,
			PerRepoEnabled: cfg.PerRepoEnabled,
			ListRepos: func() ([]store.RepoSummary, error) {
				return st.ListRepos("name", "asc")
			},
		})
		dom.RefreshStoreGauges()
	}
	return reg, dom
}

func registerBuildAndRuntime(reg prometheus.Registerer) {
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
}

func newMetricsRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	registerBuildAndRuntime(reg)
	return reg
}

func metricsExporter(reg prometheus.Gatherer) http.Handler {
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

func metricsScrapeHandler(reg prometheus.Gatherer, dom *metrics.Domain) http.Handler {
	inner := metricsExporter(reg)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if dom != nil {
			dom.RefreshStoreGauges()
		}
		inner.ServeHTTP(w, r)
	})
}

// wrapWithHTTPMetrics records request counts and latencies with a low-cardinality
// route label. Returns the wrapped handler.
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

// ServerMetrics holds the per-middleware counter vectors for rate limiter,
// whitelist, and badge endpoint.
type ServerMetrics struct {
	RateLimitedRequests *prometheus.CounterVec
	WhitelistRequests   *prometheus.CounterVec
	BadgeRequests       *prometheus.CounterVec
	BadgeDuration       *prometheus.HistogramVec
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
	case p == "/theme/custom.css":
		return "theme_custom_css"
	case p == "/":
		return "index"
	case p == "/h2h":
		return "h2h"
	}
	parts := strings.Split(strings.Trim(p, "/"), "/")
	if len(parts) == 2 && parts[0] != "" && parts[0] != "static" && parts[0] != "api" && parts[0] != "theme" {
		return "repo"
	}
	return "other"
}

// initMiddlewareMetrics creates and registers the per-middleware metric vectors
// (rate limiter, whitelist, badge) and returns them for injection into the
// respective middleware/handler wrappers. Must be called before mounting routes
// so the badge handler can record its own metrics.
func initMiddlewareMetrics(reg prometheus.Registerer) *ServerMetrics {
	rateLimitedReq := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gghstats_rate_limited_requests_total",
		Help: "Rate-limited HTTP requests by outcome (allowed = passed, blocked = 429).",
	}, []string{"status"})
	whitelistReq := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gghstats_whitelist_requests_total",
		Help: "Whitelist decisions by outcome (allowed = passed, blocked = 403).",
	}, []string{"status"})
	badgeReq := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gghstats_badge_requests_total",
		Help: "Badge SVG requests by outcome (ok, not_found, error, unauthorized, bad_request).",
	}, []string{"status"})
	badgeDur := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gghstats_badge_duration_seconds",
		Help:    "Badge SVG request latency in seconds.",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
	}, []string{"status"})
	reg.MustRegister(rateLimitedReq, whitelistReq, badgeReq, badgeDur)
	return &ServerMetrics{
		RateLimitedRequests: rateLimitedReq,
		WhitelistRequests:   whitelistReq,
		BadgeRequests:       badgeReq,
		BadgeDuration:       badgeDur,
	}
}

// wrapBadgeWithMetrics records latency and outcome for badge SVG requests.
func wrapBadgeWithMetrics(next http.HandlerFunc, cnt *prometheus.CounterVec, dur *prometheus.HistogramVec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sr, r)
		d := time.Since(start).Seconds()
		outcome := badgeOutcome(sr.status)
		cnt.WithLabelValues(outcome).Inc()
		dur.WithLabelValues(outcome).Observe(d)
	}
}

// badgeOutcome maps an HTTP status to a badge metric label.
func badgeOutcome(status int) string {
	switch {
	case status == http.StatusOK:
		return "ok"
	case status == http.StatusNotFound:
		return "not_found"
	case status == http.StatusUnauthorized:
		return "unauthorized"
	case status == http.StatusBadRequest:
		return "bad_request"
	case status >= 500:
		return "error"
	default:
		return "error"
	}
}
