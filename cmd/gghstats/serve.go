package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hrodrig/gghstats/internal/alert"
	"github.com/hrodrig/gghstats/internal/collector"
	"github.com/hrodrig/gghstats/internal/demo"
	"github.com/hrodrig/gghstats/internal/github"
	"github.com/hrodrig/gghstats/internal/i18n"
	"github.com/hrodrig/gghstats/internal/metrics"
	"github.com/hrodrig/gghstats/internal/server"
	"github.com/hrodrig/gghstats/internal/store"
	"github.com/hrodrig/gghstats/internal/sync"
	"github.com/hrodrig/gghstats/internal/version"
	"github.com/prometheus/client_golang/prometheus"
)

type serveConfig struct {
	GithubToken         string
	DB                  string
	Host                string
	Port                string
	Filter              string
	IncludePrivate      bool
	ReportPrivate       bool
	APIToken            string
	SyncInterval        time.Duration
	SyncOnStartup       bool
	SyncWorkers         int
	OpenBrowser         bool
	BadgePublic         bool
	BadgeCacheMaxAge    int
	PublicURL           string
	HeadHTML            string
	ReverseProxyRules   string
	EnableCollector     bool
	EnableUpdateCheck   bool
	Demo                bool
	APIOnly             bool
	CompactNumbers      bool
	CORSOrigins         string
	CSPMode             string
	UpstreamStaleDays   int
	UpstreamStaleBanner bool
}

func loadServeConfig() serveConfig {
	cfg := serveConfig{
		GithubToken:      os.Getenv("GGHSTATS_GITHUB_TOKEN"),
		DB:               envOr("GGHSTATS_DB", defaultDBPath()),
		Host:             envOr("GGHSTATS_HOST", "127.0.0.1"),
		Port:             envOr("GGHSTATS_PORT", "8080"),
		Filter:           envOr("GGHSTATS_FILTER", "*"),
		IncludePrivate:   envBool("GGHSTATS_INCLUDE_PRIVATE", false),
		ReportPrivate:    envBool("GGHSTATS_REPORT_PRIVATE", false),
		APIToken:         os.Getenv("GGHSTATS_API_TOKEN"),
		SyncInterval:     1 * time.Hour,
		SyncOnStartup:    envBool("GGHSTATS_SYNC_ON_STARTUP", true),
		SyncWorkers:      4,
		OpenBrowser:      envBool("GGHSTATS_OPEN_BROWSER", false),
		BadgePublic:      envBool("GGHSTATS_BADGE_PUBLIC", true),
		BadgeCacheMaxAge: 300,
		PublicURL:        strings.TrimSpace(os.Getenv("GGHSTATS_PUBLIC_URL")),
		Demo:             envBool("GGHSTATS_DEMO", false),
	}

	if v := os.Getenv("GGHSTATS_BADGE_CACHE_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.BadgeCacheMaxAge = n
		}
	}

	if v := os.Getenv("GGHSTATS_SYNC_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.SyncInterval = d
		}
	}

	if v := os.Getenv("GGHSTATS_SYNC_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.SyncWorkers = n
		}
	}

	cfg.HeadHTML = os.Getenv("GGHSTATS_HEAD_HTML")
	cfg.ReverseProxyRules = os.Getenv("GGHSTATS_REVERSE_PROXY_RULES")
	cfg.EnableCollector = envBool("GGHSTATS_ENABLE_COLLECTOR", false)
	cfg.EnableUpdateCheck = envBool("GGHSTATS_ENABLE_UPDATE_CHECK", true)
	cfg.APIOnly = envBool("GGHSTATS_API_ONLY", false)
	cfg.CompactNumbers = envBool("GGHSTATS_COMPACT_NUMBERS", false)
	cfg.CORSOrigins = os.Getenv("GGHSTATS_CORS_ORIGINS")
	cfg.CSPMode = strings.TrimSpace(os.Getenv("GGHSTATS_CSP"))
	cfg.UpstreamStaleDays = envUpstreamStaleDays()
	cfg.UpstreamStaleBanner = envBool("GGHSTATS_UPSTREAM_STALE_BANNER", true)

	return cfg
}

// errServeHelp is returned by parseServeFlags when -h/--help is requested.
// runServe treats it as a successful no-op so the help banner prints once
// and the process exits without starting the HTTP server, scheduler, or
// collector (otherwise the test/CLI would block in serveHTTP until ctx cancel).
var errServeHelp = errors.New("serve help requested")

func parseServeFlags(cfg *serveConfig, args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: gghstats serve [flags]\n\n")
		fs.PrintDefaults()
		fmt.Fprintf(fs.Output(), "\nEnvironment variables apply when flags are omitted (see: gghstats --help).\n")
	}
	fs.StringVar(&cfg.Port, "port", cfg.Port, "HTTP listen port (overrides `GGHSTATS_PORT`; default 8080)")
	fs.IntVar(&cfg.SyncWorkers, "sync-workers", cfg.SyncWorkers, "Concurrent repos per sync cycle (overrides `GGHSTATS_SYNC_WORKERS`; default 4)")
	fs.BoolVar(&cfg.OpenBrowser, "open", cfg.OpenBrowser, "Open the default browser when the server is ready")
	fs.BoolVar(&cfg.Demo, "demo", cfg.Demo, "Run with sample data; no GitHub token (overrides `GGHSTATS_DEMO`)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return errServeHelp
		}
		return err
	}
	if cfg.Demo {
		cfg.SyncOnStartup = false
		cfg.EnableUpdateCheck = false
		return nil
	}
	if cfg.GithubToken == "" {
		return fmt.Errorf("GGHSTATS_GITHUB_TOKEN is required (or use --demo / GGHSTATS_DEMO=true)")
	}
	return nil
}

func logServerStartup(cfg serveConfig) {
	writeServeStartupBanner(os.Stderr, cfg)
	initServeLogging()
	lvl := "info"
	if v := os.Getenv("GGHSTATS_LOG_LEVEL"); v != "" {
		lvl = v
	}
	listenAddr := cfg.Host + ":" + cfg.Port
	metricsEnabled := envBool("GGHSTATS_METRICS", true)
	slog.Info(fmt.Sprintf("gghstats v%s starting on %s (db=%s, filter=%s, sync=%s, log=%s, metrics=%s)",
		version.Version,
		listenAddr,
		cfg.DB,
		cfg.Filter,
		cfg.SyncInterval,
		lvl,
		map[bool]string{true: "enabled", false: "disabled"}[metricsEnabled],
	))
}

func startCollector(cfg serveConfig) {
	helpURL := "https://github.com/hrodrig/gghstats/"
	if cfg.EnableCollector && cfg.EnableUpdateCheck {
		slog.Info("Anonymous metric collection and update check are enabled. " +
			"For details about collected data see " + helpURL + " " +
			"Thank you for supporting the gghstats project!")
		go collector.CollectWithUpdate(collectFeatures(cfg))
	} else if cfg.EnableCollector {
		slog.Info("Anonymous metric collection is enabled. " +
			"For details about collected data see " + helpURL + " " +
			"Thank you for supporting the gghstats project!")
		go collector.Collect(collectFeatures(cfg))
	} else if cfg.EnableUpdateCheck {
		slog.Info("Update check is enabled.")
		slog.Info("Anonymous metric collection is disabled. " +
			"Set GGHSTATS_ENABLE_COLLECTOR=true to help improve gghstats. " +
			"See " + helpURL)
		go collector.CheckUpdate()
	} else {
		slog.Info("Anonymous metric collection is disabled. " +
			"Set GGHSTATS_ENABLE_COLLECTOR=true to help improve gghstats. " +
			"See " + helpURL)
	}
}

// loadAlertSenders validates GGHSTATS_ALERTS_* (SPEC §8.5 fail-closed) and builds senders.
// Returns nil senders when alerts are disabled.
func loadAlertSenders() ([]alert.Sender, error) {
	enabled, sinks, err := alert.ConfigFromEnv(os.Getenv)
	if err != nil {
		return nil, fmt.Errorf("alert config: %w", err)
	}
	if !enabled {
		return nil, nil
	}
	senders := alert.BuildSenders(sinks, nil)
	slog.Info(fmt.Sprintf("alerts enabled: %d sink(s) configured", len(senders)))
	return senders, nil
}

func collectFeatures(cfg serveConfig) collector.ServeFeatures {
	return collector.ServeFeatures{
		BadgePublic:      cfg.BadgePublic,
		MetricsEnabled:   envBool("GGHSTATS_METRICS", true),
		MetricsPerRepo:   envBool("GGHSTATS_METRICS_PER_REPO", false),
		SyncOnStartup:    cfg.SyncOnStartup,
		HasAPIToken:      cfg.APIToken != "",
		HasPublicURL:     cfg.PublicURL != "",
		HasCustomCSS:     os.Getenv("GGHSTATS_CUSTOM_CSS") != "",
		RateLimitEnabled: server.ParseRateLimitEnv().Enabled,
		Port:             cfg.Port,
		Host:             cfg.Host,
	}
}

func runServe(args []string) error {
	cfg := loadServeConfig()

	if err := parseServeFlags(&cfg, args); err != nil {
		if errors.Is(err, errServeHelp) {
			return nil
		}
		return err
	}

	logServerStartup(cfg)

	alertSenders, err := loadAlertSenders()
	if err != nil {
		return err
	}
	alertRules, err := alert.RulesFromEnv(os.Getenv)
	if err != nil {
		return fmt.Errorf("alert rules: %w", err)
	}

	db, err := store.Open(cfg.DB)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := seedDemoIfEnabled(db, cfg.Demo); err != nil {
		return err
	}
	initialSyncPending, settingsManager, editableSettings, enabledLocales, err := loadEditableRuntimeSettings(cfg, db)
	if err != nil {
		return err
	}

	metricsReg, domainMetrics := setupMetrics(cfg, db)

	rateLimiter := setupRateLimiter()
	if rateLimiter != nil {
		defer rateLimiter.Shutdown()
	}

	trusted := server.ParseTrustedProxies(os.Getenv("GGHSTATS_TRUSTED_PROXIES"))
	whitelist := server.NewWhitelist(server.ParseWhitelistEnv(), cfg.APIToken)
	server.WarnTrustedProxiesIfNeeded(trusted, rateLimiter != nil, whitelist != nil)
	corsOrigins := server.ParseCORSOrigins(cfg.CORSOrigins)
	warnAPIOnlyOpenCORS(cfg.APIOnly, corsOrigins)
	warnCSPEnforceWithHeadHTML(cfg.CSPMode, cfg.HeadHTML)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	coord := setupSyncCoordinator(ctx, cfg, db, domainMetrics, alertSenders, alertRules)

	cssAbs, cssQuery := resolveCSSPath()

	// Start HTTP server
	handler := server.New(server.Config{
		Store:               db,
		ReportVisibility:    store.ReportVisibility{IncludePrivate: cfg.ReportPrivate},
		APIToken:            cfg.APIToken,
		SyncCoordinator:     coord,
		SyncOnStartup:       cfg.SyncOnStartup,
		InitialSyncPending:  initialSyncPending,
		BadgePublic:         cfg.BadgePublic,
		BadgeCacheMaxAge:    cfg.BadgeCacheMaxAge,
		PublicURL:           cfg.PublicURL,
		DisableMetrics:      !envBool("GGHSTATS_METRICS", true),
		MetricsRegistry:     metricsReg,
		DomainMetrics:       domainMetrics,
		CustomCSSAbsPath:    cssAbs,
		CustomCSSQuery:      cssQuery,
		DefaultLocale:       editableSettings.DefaultLocale,
		EnabledLocales:      enabledLocales,
		CompactNumbers:      editableSettings.CompactNumbers,
		UpstreamStaleDays:   cfg.UpstreamStaleDays,
		UpstreamStaleBanner: cfg.UpstreamStaleBanner,
		SettingsManager:     settingsManager,
		LocalOnlySettings:   isLoopbackBindHost(cfg.Host),
		RateLimiter:         rateLimiter,
		TrustedProxies:      trusted,
		Whitelist:           whitelist,
		HeadHTML:            template.HTML(cfg.HeadHTML),
		ReverseProxyRules:   server.ParseReverseProxyRules(cfg.ReverseProxyRules),
		APIOnly:             cfg.APIOnly,
		CORSOrigins:         corsOrigins,
		CSPMode:             cfg.CSPMode,
		Settings: server.SettingsSnapshot{
			RuntimeMode:              map[bool]string{true: "Demo", false: "Production"}[cfg.Demo],
			Host:                     cfg.Host,
			Port:                     cfg.Port,
			DatabaseReady:            db != nil,
			GithubTokenConfigured:    cfg.GithubToken != "",
			APITokenConfigured:       cfg.APIToken != "",
			FilterAll:                strings.TrimSpace(cfg.Filter) == "" || strings.TrimSpace(cfg.Filter) == "*",
			IncludePrivate:           cfg.IncludePrivate,
			ReportPrivate:            cfg.ReportPrivate,
			SyncInterval:             cfg.SyncInterval.String(),
			SyncOnStartup:            cfg.SyncOnStartup,
			SyncWorkers:              cfg.SyncWorkers,
			SyncAvailable:            coord != nil,
			BadgePublic:              cfg.BadgePublic,
			MetricsEnabled:           envBool("GGHSTATS_METRICS", true),
			RateLimitEnabled:         rateLimiter != nil,
			WhitelistEnabled:         whitelist != nil,
			TrustedProxiesConfigured: trusted.Configured(),
			CORSConfigured:           len(corsOrigins) > 0,
			CSPEnforced:              strings.EqualFold(strings.TrimSpace(cfg.CSPMode), "enforce") && strings.TrimSpace(cfg.HeadHTML) == "",
			CustomCSSConfigured:      cssAbs != "",
			ReverseProxyConfigured:   len(server.ParseReverseProxyRules(cfg.ReverseProxyRules)) > 0,
			AlertsConfigured:         len(alertSenders) > 0 && len(alertRules) > 0,
			APIOnly:                  cfg.APIOnly,
			CompactNumbers:           editableSettings.CompactNumbers,
			DefaultLocale:            editableSettings.DefaultLocale,
			EnabledLocales:           enabledLocales,
		},
	})

	startCollector(cfg)

	addr := cfg.Host + ":" + cfg.Port
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return serveHTTP(ctx, srv, cfg, cancel)
}

func loadEditableRuntimeSettings(cfg serveConfig, db *store.Store) (bool, *server.SettingsManager, server.EditableSettings, []string, error) {
	initialSyncPending := false
	if !cfg.Demo && cfg.SyncOnStartup {
		visibleRepos, err := db.ReportRepoCount(store.ReportVisibility{IncludePrivate: cfg.ReportPrivate})
		if err != nil {
			return false, nil, server.EditableSettings{}, nil, fmt.Errorf("check initial database state: %w", err)
		}
		initialSyncPending = visibleRepos == 0
	}

	enabledLocales := i18n.EnvEnabledLocales()
	settingsPath := ""
	if cfg.DB != ":memory:" {
		settingsPath = cfg.DB + ".settings.json"
	}
	settingsManager, err := server.NewSettingsManager(settingsPath, server.EditableSettings{
		DefaultLocale:  i18n.EnvDefaultLocale(),
		CompactNumbers: cfg.CompactNumbers,
	}, enabledLocales)
	if err != nil {
		return false, nil, server.EditableSettings{}, nil, fmt.Errorf("load settings: %w", err)
	}
	return initialSyncPending, settingsManager, settingsManager.Snapshot(), enabledLocales, nil
}

func setupMetrics(cfg serveConfig, db *store.Store) (*prometheus.Registry, *metrics.Domain) {
	if !envBool("GGHSTATS_METRICS", true) {
		return nil, nil
	}
	return server.NewMetricsRegistry(server.MetricsRegistryConfig{
		Store:            db,
		DBPath:           cfg.DB,
		PerRepoEnabled:   envBool("GGHSTATS_METRICS_PER_REPO", false),
		ReportVisibility: store.ReportVisibility{IncludePrivate: cfg.ReportPrivate},
	})
}

func setupSyncCoordinator(ctx context.Context, cfg serveConfig, db *store.Store, domainMetrics *metrics.Domain, alertSenders []alert.Sender, alertRules []alert.RuleSpec) *sync.Coordinator {
	if cfg.Demo {
		slog.Info("demo mode: GitHub sync and update check disabled")
		return nil
	}

	gh := github.NewClient(cfg.GithubToken)
	applyOptionalGitHubBaseURL(gh)
	if domainMetrics != nil {
		gh.SetMetrics(domainMetrics)
	}
	coord := sync.NewCoordinator(gh, db, sync.Options{
		IncludePrivate: cfg.IncludePrivate,
		Filter:         cfg.Filter,
		SyncStars:      true,
		Workers:        cfg.SyncWorkers,
	})
	if domainMetrics != nil {
		coord.SetMetrics(domainMetrics)
	}
	if len(alertSenders) > 0 && len(alertRules) > 0 {
		configureSyncAlerts(ctx, cfg, db, coord, alertSenders, alertRules)
	}
	go startScheduler(ctx, coord, cfg.SyncInterval, cfg.SyncOnStartup)
	return coord
}

func configureSyncAlerts(ctx context.Context, cfg serveConfig, db *store.Store, coord *sync.Coordinator, senders []alert.Sender, rules []alert.RuleSpec) {
	publicURL := cfg.PublicURL
	coord.SetAfterSync(func(result sync.RunResult) {
		reportScope := store.ReportVisibility{IncludePrivate: cfg.ReportPrivate}
		failed := reportableFailedRepos(db, reportScope, result.FailedRepos)
		attempted, _ := db.ReportRepoCount(reportScope)
		snap := alert.SyncSnapshot{
			// Report alerts must not disclose failures from excluded collection.
			Success:            result.Success || len(failed) == 0,
			ReposAttempted:     attempted,
			ReposFailed:        len(failed),
			FailedRepos:        failed,
			Unreachable:        result.Unreachable,
			RateLimitRemaining: result.RateLimitRemaining,
		}
		ustale, err := store.DetectFleetUpstreamStale(db, reportScope, cfg.UpstreamStaleDays, time.Now().UTC(), snap.Success)
		if err != nil {
			slog.Error("alerts: detect upstream stale", "error", err)
		} else {
			snap.UpstreamStale = ustale.Active
			snap.UpstreamStaleSince = ustale.Since
		}
		alert.RunAllRules(ctx, alert.EvalConfig{
			DB:               db,
			ReportVisibility: reportScope,
			Rules:            rules,
			Senders:          senders,
			PublicURL:        publicURL,
		}, snap)
	})
	slog.Info(fmt.Sprintf("alerts: %d rule(s) will evaluate after sync", len(rules)))
}

func seedDemoIfEnabled(db *store.Store, enabled bool) error {
	if !enabled {
		return nil
	}
	if err := demo.SeedIfEmpty(db); err != nil {
		return fmt.Errorf("demo seed: %w", err)
	}
	return nil
}

func isLoopbackBindHost(host string) bool {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func reportableFailedRepos(db *store.Store, scope store.ReportVisibility, names []string) []string {
	visible := make([]string, 0, len(names))
	for _, name := range names {
		repo, err := db.ReportRepoByName(scope, name)
		if err == nil && repo != nil {
			visible = append(visible, name)
		}
	}
	return visible
}

// serveHTTP listens until ctx is cancelled or ListenAndServe fails.
// cancel stops the sync scheduler; callers must still defer their own cancel.
func serveHTTP(ctx context.Context, srv *http.Server, cfg serveConfig, cancel context.CancelFunc) error {
	errCh := make(chan error, 1)
	go func() {
		logURL := serveDashboardURL(cfg.Host, cfg.Port)
		if cfg.Host == "0.0.0.0" || cfg.Host == "::" || cfg.Host == "[::]" {
			slog.Info("listening", "url", logURL, "bind", srv.Addr)
		} else {
			slog.Info("listening", "url", logURL)
		}
		if cfg.OpenBrowser {
			go func() {
				time.Sleep(300 * time.Millisecond)
				openBrowser(serveOpenURL(cfg.Host, cfg.Port, cfg.APIOnly))
			}()
		}
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		cancel()
		return err
	case <-ctx.Done():
		slog.Info("shutting down...")
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			<-errCh
			return err
		}
		return <-errCh
	}
}

func startScheduler(ctx context.Context, coord *sync.Coordinator, interval time.Duration, syncOnStartup bool) {
	if syncOnStartup {
		// Optional: full sync on startup so discovery matches the current filter without waiting for the interval.
		slog.Info("startup sync starting")
		if err := coord.Run(); err != nil {
			if errors.Is(err, sync.ErrInProgress) {
				slog.Warn("startup sync skipped", "reason", err)
			} else {
				slog.Error("startup sync failed", "error", err)
			}
		}
	} else {
		slog.Info("startup sync disabled", "hint", "set GGHSTATS_SYNC_ON_STARTUP=true or use POST /api/v1/sync")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduler stopped")
			return
		case <-ticker.C:
			slog.Info("scheduled sync starting")
			if err := coord.Run(); err != nil {
				if errors.Is(err, sync.ErrInProgress) {
					slog.Info("scheduled sync skipped", "reason", err)
				} else {
					slog.Error("scheduled sync failed", "error", err)
				}
			}
		}
	}
}

func resolveCSSPath() (cssAbs, cssQuery string) {
	cssAbs, cssQuery = server.ResolveCustomCSS(os.Getenv("GGHSTATS_CUSTOM_CSS"))
	if strings.TrimSpace(os.Getenv("GGHSTATS_CUSTOM_CSS")) != "" && cssAbs == "" {
		slog.Warn("GGHSTATS_CUSTOM_CSS ignored: path is missing or not a regular file",
			"GGHSTATS_CUSTOM_CSS", os.Getenv("GGHSTATS_CUSTOM_CSS"))
	}
	return
}

func setupRateLimiter() *server.RateLimiter {
	rlCfg := server.ParseRateLimitEnv()
	if !rlCfg.Enabled {
		return nil
	}
	rl := server.NewRateLimiter(rlCfg)
	slog.Info("rate limiter enabled",
		"requests", rlCfg.Requests,
		"period", rlCfg.Period,
		"burst", rlCfg.Burst,
	)
	return rl
}

func warnAPIOnlyOpenCORS(apiOnly bool, origins []string) {
	if apiOnly && server.CORSIsOpen(origins) {
		slog.Warn("API-only mode with open CORS (*): browser clients can call the API from any origin; do not embed GGHSTATS_API_TOKEN in a public SPA — use a BFF/proxy or set GGHSTATS_CORS_ORIGINS")
	}
}

func warnCSPEnforceWithHeadHTML(cspMode, headHTML string) {
	if strings.EqualFold(strings.TrimSpace(cspMode), "enforce") && strings.TrimSpace(headHTML) != "" {
		slog.Warn("GGHSTATS_CSP=enforce ignored while GGHSTATS_HEAD_HTML is set; using Content-Security-Policy-Report-Only")
	}
}
