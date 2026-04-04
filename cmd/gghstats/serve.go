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
	"syscall"
	"time"

	"github.com/hrodrig/gghstats/internal/github"
	"github.com/hrodrig/gghstats/internal/server"
	"github.com/hrodrig/gghstats/internal/store"
	"github.com/hrodrig/gghstats/internal/sync"
)

type serveConfig struct {
	GithubToken    string
	DB             string
	Host           string
	Port           string
	Filter         string
	IncludePrivate bool
	APIToken       string
	SyncInterval   time.Duration
}

func loadServeConfig() serveConfig {
	cfg := serveConfig{
		GithubToken:    os.Getenv("GGHSTATS_GITHUB_TOKEN"),
		DB:             envOr("GGHSTATS_DB", "./data/gghstats.db"),
		Host:           envOr("GGHSTATS_HOST", "0.0.0.0"),
		Port:           envOr("GGHSTATS_PORT", "8080"),
		Filter:         envOr("GGHSTATS_FILTER", "*"),
		IncludePrivate: os.Getenv("GGHSTATS_INCLUDE_PRIVATE") == "true",
		APIToken:       os.Getenv("GGHSTATS_API_TOKEN"),
		SyncInterval:   1 * time.Hour,
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
	)

	db, err := store.Open(cfg.DB)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	gh := github.NewClient(cfg.GithubToken)
	applyOptionalGitHubBaseURL(gh)

	syncOpts := sync.Options{
		IncludePrivate: cfg.IncludePrivate,
		Filter:         cfg.Filter,
		SyncStars:      true,
	}

	// Start scheduler in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go startScheduler(ctx, gh, db, syncOpts, cfg.SyncInterval)

	// Start HTTP server
	handler := server.New(server.Config{
		Store:          db,
		APIToken:       cfg.APIToken,
		DisableMetrics: os.Getenv("GGHSTATS_METRICS") == "false",
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
		slog.Info("listening", "addr", "http://"+addr)
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

func startScheduler(ctx context.Context, gh *github.Client, db *store.Store, opts sync.Options, interval time.Duration) {
	// Full sync on startup so repo discovery matches the current filter without waiting for the interval.
	slog.Info("startup sync starting")
	if err := sync.Run(gh, db, opts); err != nil {
		slog.Error("startup sync failed", "error", err)
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
			if err := sync.Run(gh, db, opts); err != nil {
				slog.Error("scheduled sync failed", "error", err)
			}
		}
	}
}
