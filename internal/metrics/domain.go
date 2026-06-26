package metrics

import (
	"os"
	"sync"
	"time"

	"github.com/hrodrig/gghstats/internal/store"
	"github.com/prometheus/client_golang/prometheus"
)

// Domain holds Prometheus metrics for sync, GitHub API, and store state.
type Domain struct {
	mu sync.Mutex

	filter string
	dbPath string

	storeRepoCount func() (int, error)
	perRepoEnabled bool
	listRepos      func() ([]store.RepoSummary, error)

	githubRequests *prometheus.CounterVec
	githubRateRem  *prometheus.GaugeVec
	syncDuration   *prometheus.HistogramVec
	syncErrors     *prometheus.CounterVec
	lastSyncTS     prometheus.Gauge
	reposTotal     *prometheus.GaugeVec
	dbSizeBytes    prometheus.Gauge

	repoStars      *prometheus.GaugeVec
	repoForks      *prometheus.GaugeVec
	repoClones     *prometheus.GaugeVec
	repoViews      *prometheus.GaugeVec
	repoClones1d   *prometheus.GaugeVec
	repoClones7d   *prometheus.GaugeVec
	repoClones30d  *prometheus.GaugeVec
	lastRepoLabels map[string]struct{}
}

// DomainConfig wires domain metrics at process startup.
type DomainConfig struct {
	Filter         string
	DBPath         string
	StoreRepoCount func() (int, error) // nil skips repos_total refresh
	PerRepoEnabled bool
	ListRepos      func() ([]store.RepoSummary, error) // required when PerRepoEnabled
}

// RegisterDomain registers domain collectors on reg and returns the recorder.
func RegisterDomain(reg prometheus.Registerer, cfg DomainConfig) *Domain {
	filter := cfg.Filter
	if filter == "" {
		filter = "*"
	}

	d := &Domain{
		filter:         filter,
		dbPath:         cfg.DBPath,
		storeRepoCount: cfg.StoreRepoCount,
		perRepoEnabled: cfg.PerRepoEnabled,
		listRepos:      cfg.ListRepos,
		githubRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gghstats_github_api_requests_total",
				Help: "GitHub REST API requests by normalized endpoint and outcome.",
			},
			[]string{"endpoint", "status"},
		),
		githubRateRem: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gghstats_github_rate_limit_remaining",
				Help: "GitHub REST rate limit remaining (from X-RateLimit-Remaining).",
			},
			[]string{"resource"},
		),
		syncDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gghstats_sync_duration_seconds",
				Help:    "Duration of a full sync cycle.",
				Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600, 1200, 1800, 3600},
			},
			[]string{"status"},
		),
		syncErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gghstats_sync_errors_total",
				Help: "Sync errors by classification kind (worker = repo-level failure).",
			},
			[]string{"kind"},
		),
		lastSyncTS: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "gghstats_last_sync_timestamp_seconds",
			Help: "Unix timestamp of the last successful full sync.",
		}),
		reposTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gghstats_repos_total",
				Help: "Number of non-hidden repositories in the local database.",
			},
			[]string{"filter"},
		),
		dbSizeBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "gghstats_db_size_bytes",
			Help: "Size of the SQLite database file on disk.",
		}),
	}

	toRegister := []prometheus.Collector{
		d.githubRequests,
		d.githubRateRem,
		d.syncDuration,
		d.syncErrors,
		d.lastSyncTS,
		d.reposTotal,
		d.dbSizeBytes,
	}
	if cfg.PerRepoEnabled {
		d.repoStars = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gghstats_repo_stars",
			Help: "Repository star count (latest sync).",
		}, []string{"owner", "repo"})
		d.repoForks = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gghstats_repo_forks",
			Help: "Repository fork count (latest sync).",
		}, []string{"owner", "repo"})
		d.repoClones = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gghstats_repo_clones",
			Help: "Lifetime clone count in SQLite (sum of daily rows).",
		}, []string{"owner", "repo"})
		d.repoViews = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gghstats_repo_views",
			Help: "Lifetime view count in SQLite (sum of daily rows).",
		}, []string{"owner", "repo"})
		d.repoClones1d = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gghstats_repo_clones_1d",
			Help: "Clone count for latest UTC day with data (today or yesterday).",
		}, []string{"owner", "repo"})
		d.repoClones7d = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gghstats_repo_clones_7d",
			Help: "Sum of daily clone counts over the last 7 UTC calendar days.",
		}, []string{"owner", "repo"})
		d.repoClones30d = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gghstats_repo_clones_30d",
			Help: "Sum of daily clone counts over the last 30 UTC calendar days.",
		}, []string{"owner", "repo"})
		d.lastRepoLabels = make(map[string]struct{})
		toRegister = append(toRegister,
			d.repoStars, d.repoForks, d.repoClones, d.repoViews,
			d.repoClones1d, d.repoClones7d, d.repoClones30d,
		)
	}
	reg.MustRegister(toRegister...)
	return d
}

// ObserveGitHubRequest records one GitHub API call.
func (d *Domain) ObserveGitHubRequest(endpoint, status string) {
	if d == nil {
		return
	}
	d.githubRequests.WithLabelValues(endpoint, status).Inc()
}

// SetGitHubRateLimitRemaining updates the core REST rate limit gauge.
func (d *Domain) SetGitHubRateLimitRemaining(remaining int) {
	if d == nil {
		return
	}
	d.githubRateRem.WithLabelValues("core").Set(float64(remaining))
}

// ObserveSync records sync duration and last-success timestamp.
func (d *Domain) ObserveSync(duration time.Duration, success bool) {
	if d == nil {
		return
	}
	status := "success"
	if !success {
		status = "error"
	}
	d.syncDuration.WithLabelValues(status).Observe(duration.Seconds())
	if success {
		d.lastSyncTS.Set(float64(time.Now().UTC().Unix()))
		d.RefreshStoreGauges()
	}
}

// ObserveSyncError increments the per-kind sync error counter. kind is a
// short classifier such as "worker", "repo_meta", "views", "clones",
// "referrers", "paths", "stars", or "stargazers".
func (d *Domain) ObserveSyncError(kind string) {
	if d == nil || kind == "" {
		return
	}
	d.syncErrors.WithLabelValues(kind).Inc()
}

// RefreshStoreGauges updates repos_total, db_size_bytes, and optional per-repo gauges.
func (d *Domain) RefreshStoreGauges() {
	if d == nil {
		return
	}
	d.mu.Lock()
	if d.storeRepoCount != nil {
		if n, err := d.storeRepoCount(); err == nil {
			d.reposTotal.WithLabelValues(d.filter).Set(float64(n))
		}
	}
	if d.dbPath != "" {
		if fi, err := os.Stat(d.dbPath); err == nil {
			d.dbSizeBytes.Set(float64(fi.Size()))
		}
	}
	d.mu.Unlock()
	d.refreshPerRepoGauges()
}
