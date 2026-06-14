package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hrodrig/gghstats/internal/github"
	"github.com/hrodrig/gghstats/internal/i18n"
	"github.com/hrodrig/gghstats/internal/metrics"
	"github.com/hrodrig/gghstats/internal/server"
	"github.com/hrodrig/gghstats/internal/store"
	"github.com/hrodrig/gghstats/internal/sync"
	"github.com/prometheus/client_golang/prometheus"
)

type serveConfig struct {
	GithubToken      string
	DB               string
	Host             string
	Port             string
	Filter           string
	IncludePrivate   bool
	APIToken         string
	SyncInterval     time.Duration
	SyncOnStartup    bool
	OpenBrowser      bool
	BadgePublic      bool
	BadgeCacheMaxAge int
	PublicURL        string
}

func loadServeConfig() serveConfig {
	cfg := serveConfig{
		GithubToken:      os.Getenv("GGHSTATS_GITHUB_TOKEN"),
		DB:               envOr("GGHSTATS_DB", "./data/gghstats.db"),
		Host:             envOr("GGHSTATS_HOST", "127.0.0.1"),
		Port:             envOr("GGHSTATS_PORT", "8080"),
		Filter:           envOr("GGHSTATS_FILTER", "*"),
		IncludePrivate:   os.Getenv("GGHSTATS_INCLUDE_PRIVATE") == "true",
		APIToken:         os.Getenv("GGHSTATS_API_TOKEN"),
		SyncInterval:     1 * time.Hour,
		SyncOnStartup:    envBool("GGHSTATS_SYNC_ON_STARTUP", true),
		OpenBrowser:      envBool("GGHSTATS_OPEN_BROWSER", false),
		BadgePublic:      os.Getenv("GGHSTATS_BADGE_PUBLIC") != "false",
		BadgeCacheMaxAge: 300,
		PublicURL:        strings.TrimSpace(os.Getenv("GGHSTATS_PUBLIC_URL")),
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

	return cfg
}

// envOrDefault is defined in flags.go as envOr

func runServe(args []string) error {
	cfg := loadServeConfig()

	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: gghstats serve [flags]\n\n")
		fs.PrintDefaults()
		fmt.Fprintf(fs.Output(), "\nEnvironment variables apply when flags are omitted (see: gghstats --help).\n")
	}
	fs.StringVar(&cfg.Port, "port", cfg.Port, "HTTP listen port (overrides `GGHSTATS_PORT`; default 8080)")
	fs.BoolVar(&cfg.OpenBrowser, "open", cfg.OpenBrowser, "Open the default browser when the server is ready")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if cfg.GithubToken == "" {
		return fmt.Errorf("GGHSTATS_GITHUB_TOKEN is required")
	}

	writeServeStartupBanner(os.Stderr, cfg)
	initServeLogging()

	slog.Info("gghstats starting",
		"db", cfg.DB,
		"filter", cfg.Filter,
		"sync_interval", cfg.SyncInterval,
		"sync_on_startup", cfg.SyncOnStartup,
	)

	db, err := store.Open(cfg.DB)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	gh := github.NewClient(cfg.GithubToken)
	applyOptionalGitHubBaseURL(gh)

	var metricsReg *prometheus.Registry
	var domainMetrics *metrics.Domain
	if os.Getenv("GGHSTATS_METRICS") != "false" {
		metricsReg, domainMetrics = server.NewMetricsRegistry(server.MetricsRegistryConfig{
			Store:          db,
			DBPath:         cfg.DB,
			Filter:         cfg.Filter,
			PerRepoEnabled: os.Getenv("GGHSTATS_METRICS_PER_REPO") == "true",
		})
		gh.SetMetrics(domainMetrics)
	}

	rateLimiter := setupRateLimiter()
	if rateLimiter != nil {
		defer rateLimiter.Shutdown()
	}

	whitelist := server.NewWhitelist(server.ParseWhitelistEnv())

	syncOpts := sync.Options{
		IncludePrivate: cfg.IncludePrivate,
		Filter:         cfg.Filter,
		SyncStars:      true,
	}
	coord := sync.NewCoordinator(gh, db, syncOpts)
	if domainMetrics != nil {
		coord.SetMetrics(domainMetrics)
	}

	// Start scheduler in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go startScheduler(ctx, coord, cfg.SyncInterval, cfg.SyncOnStartup)

	cssAbs, cssQuery := resolveCSSPath()

	// Start HTTP server
	handler := server.New(server.Config{
		Store:            db,
		APIToken:         cfg.APIToken,
		SyncCoordinator:  coord,
		BadgePublic:      cfg.BadgePublic,
		BadgeCacheMaxAge: cfg.BadgeCacheMaxAge,
		PublicURL:        cfg.PublicURL,
		DisableMetrics:   os.Getenv("GGHSTATS_METRICS") == "false",
		MetricsRegistry:  metricsReg,
		DomainMetrics:    domainMetrics,
		CustomCSSAbsPath: cssAbs,
		CustomCSSQuery:   cssQuery,
		DefaultLocale:    i18n.EnvDefaultLocale(),
		EnabledLocales:   i18n.EnvEnabledLocales(),
		RateLimiter:      rateLimiter,
		Whitelist:        whitelist,
	})

	addr := cfg.Host + ":" + cfg.Port
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logURL := serveDashboardURL(cfg.Host, cfg.Port)
		if cfg.Host == "0.0.0.0" || cfg.Host == "::" || cfg.Host == "[::]" {
			slog.Info("listening", "url", logURL, "bind", addr)
		} else {
			slog.Info("listening", "url", logURL)
		}
		if cfg.OpenBrowser {
			go func() {
				time.Sleep(300 * time.Millisecond)
				openBrowser(logURL)
			}()
		}
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	return srv.Shutdown(shutdownCtx)
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
