package main

import (
	"testing"
	"time"
)

func TestLoadServeConfigDefaults(t *testing.T) {
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
	t.Setenv("GGHSTATS_DB", "")
	t.Setenv("GGHSTATS_HOST", "")
	t.Setenv("GGHSTATS_PORT", "")
	t.Setenv("GGHSTATS_FILTER", "")
	t.Setenv("GGHSTATS_INCLUDE_PRIVATE", "")
	t.Setenv("GGHSTATS_API_TOKEN", "")
	t.Setenv("GGHSTATS_SYNC_INTERVAL", "")

	cfg := loadServeConfig()
	if cfg.Host != "0.0.0.0" || cfg.Port != "8080" {
		t.Fatalf("host/port: %+v", cfg)
	}
	if cfg.Filter != "*" {
		t.Errorf("Filter = %q", cfg.Filter)
	}
	if cfg.SyncInterval != time.Hour {
		t.Errorf("SyncInterval = %v", cfg.SyncInterval)
	}
}

func TestLoadServeConfigOverrides(t *testing.T) {
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "ghp_x")
	t.Setenv("GGHSTATS_DB", "/data/db.sqlite")
	t.Setenv("GGHSTATS_HOST", "127.0.0.1")
	t.Setenv("GGHSTATS_PORT", "3000")
	t.Setenv("GGHSTATS_FILTER", "org/*")
	t.Setenv("GGHSTATS_INCLUDE_PRIVATE", "true")
	t.Setenv("GGHSTATS_API_TOKEN", "api-secret")
	t.Setenv("GGHSTATS_SYNC_INTERVAL", "45m")

	cfg := loadServeConfig()
	if cfg.GithubToken != "ghp_x" || cfg.DB != "/data/db.sqlite" {
		t.Fatalf("token/db: %+v", cfg)
	}
	if cfg.Host != "127.0.0.1" || cfg.Port != "3000" {
		t.Fatalf("listen: %+v", cfg)
	}
	if cfg.Filter != "org/*" || !cfg.IncludePrivate {
		t.Fatalf("filter/private: %+v", cfg)
	}
	if cfg.APIToken != "api-secret" {
		t.Errorf("APIToken = %q", cfg.APIToken)
	}
	if cfg.SyncInterval != 45*time.Minute {
		t.Errorf("SyncInterval = %v", cfg.SyncInterval)
	}
}

func TestLoadServeConfigInvalidSyncIntervalIgnored(t *testing.T) {
	t.Setenv("GGHSTATS_SYNC_INTERVAL", "not-a-duration")
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
	t.Setenv("GGHSTATS_DB", "")
	t.Setenv("GGHSTATS_HOST", "")
	t.Setenv("GGHSTATS_PORT", "")
	t.Setenv("GGHSTATS_FILTER", "")
	t.Setenv("GGHSTATS_INCLUDE_PRIVATE", "")
	t.Setenv("GGHSTATS_API_TOKEN", "")

	cfg := loadServeConfig()
	if cfg.SyncInterval != time.Hour {
		t.Errorf("want default 1h when interval invalid, got %v", cfg.SyncInterval)
	}
}
