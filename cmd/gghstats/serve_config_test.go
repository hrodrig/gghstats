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
	t.Setenv("GGHSTATS_SYNC_ON_STARTUP", "")

	cfg := loadServeConfig()
	if cfg.Host != "127.0.0.1" || cfg.Port != "8080" {
		t.Fatalf("host/port: %+v", cfg)
	}
	if cfg.Filter != "*" {
		t.Errorf("Filter = %q", cfg.Filter)
	}
	if cfg.SyncInterval != time.Hour {
		t.Errorf("SyncInterval = %v", cfg.SyncInterval)
	}
	if !cfg.SyncOnStartup {
		t.Error("SyncOnStartup default want true")
	}
	if cfg.OpenBrowser {
		t.Error("OpenBrowser default want false")
	}
}

func TestLoadServeConfigOpenBrowser(t *testing.T) {
	t.Setenv("GGHSTATS_OPEN_BROWSER", "true")
	cfg := loadServeConfig()
	if !cfg.OpenBrowser {
		t.Error("OpenBrowser want true from env")
	}
}

func TestLoadServeConfigSyncOnStartupFalse(t *testing.T) {
	t.Setenv("GGHSTATS_SYNC_ON_STARTUP", "false")
	cfg := loadServeConfig()
	if cfg.SyncOnStartup {
		t.Error("SyncOnStartup want false")
	}
}

func TestLoadServeConfigOverrides(t *testing.T) {
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "ghp_x")
	t.Setenv("GGHSTATS_DB", "/data/db.sqlite")
	t.Setenv("GGHSTATS_HOST", "0.0.0.0")
	t.Setenv("GGHSTATS_PORT", "3000")
	t.Setenv("GGHSTATS_FILTER", "org/*")
	t.Setenv("GGHSTATS_INCLUDE_PRIVATE", "true")
	t.Setenv("GGHSTATS_API_TOKEN", "api-secret")
	t.Setenv("GGHSTATS_SYNC_INTERVAL", "45m")

	cfg := loadServeConfig()
	if cfg.GithubToken != "ghp_x" || cfg.DB != "/data/db.sqlite" {
		t.Fatalf("token/db: %+v", cfg)
	}
	if cfg.Host != "0.0.0.0" || cfg.Port != "3000" {
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

func TestLoadServeConfigCollectorAndUpdateDefaults(t *testing.T) {
	t.Setenv("GGHSTATS_ENABLE_COLLECTOR", "")
	t.Setenv("GGHSTATS_ENABLE_UPDATE_CHECK", "")

	cfg := loadServeConfig()
	if cfg.EnableCollector {
		t.Errorf("EnableCollector default want false (opt-in)")
	}
	if !cfg.EnableUpdateCheck {
		t.Errorf("EnableUpdateCheck default want true (opt-out)")
	}
}

func TestLoadServeConfigCollectorOptIn(t *testing.T) {
	t.Setenv("GGHSTATS_ENABLE_COLLECTOR", "true")

	cfg := loadServeConfig()
	if !cfg.EnableCollector {
		t.Errorf("EnableCollector want true after GGHSTATS_ENABLE_COLLECTOR=true")
	}
}

func TestLoadServeConfigUpdateCheckOptOut(t *testing.T) {
	t.Setenv("GGHSTATS_ENABLE_UPDATE_CHECK", "false")

	cfg := loadServeConfig()
	if cfg.EnableUpdateCheck {
		t.Errorf("EnableUpdateCheck want false after GGHSTATS_ENABLE_UPDATE_CHECK=false")
	}
}

func TestLoadServeConfigIncludePrivateTruthyAliases(t *testing.T) {
	for _, v := range []string{"1", "yes", "on"} {
		t.Run(v, func(t *testing.T) {
			t.Setenv("GGHSTATS_INCLUDE_PRIVATE", v)
			cfg := loadServeConfig()
			if !cfg.IncludePrivate {
				t.Errorf("IncludePrivate want true for %q", v)
			}
		})
	}
}

func TestLoadServeConfigCollectorOptInAlias(t *testing.T) {
	t.Setenv("GGHSTATS_ENABLE_COLLECTOR", "1")
	cfg := loadServeConfig()
	if !cfg.EnableCollector {
		t.Error("EnableCollector want true for GGHSTATS_ENABLE_COLLECTOR=1")
	}
}

func TestLoadServeConfigBadgePublicFalseAlias(t *testing.T) {
	t.Setenv("GGHSTATS_BADGE_PUBLIC", "0")
	cfg := loadServeConfig()
	if cfg.BadgePublic {
		t.Error("BadgePublic want false for GGHSTATS_BADGE_PUBLIC=0")
	}
}

func TestParseServeFlagsDemoSkipsToken(t *testing.T) {
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
	t.Setenv("GGHSTATS_DEMO", "")
	cfg := loadServeConfig()
	if err := parseServeFlags(&cfg, []string{"--demo"}); err != nil {
		t.Fatal(err)
	}
	if !cfg.Demo {
		t.Fatal("Demo want true")
	}
	if cfg.SyncOnStartup {
		t.Fatal("SyncOnStartup want false in demo")
	}
	if cfg.EnableUpdateCheck {
		t.Fatal("EnableUpdateCheck want false in demo")
	}
}

func TestParseServeFlagsDemoEnv(t *testing.T) {
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
	t.Setenv("GGHSTATS_DEMO", "true")
	cfg := loadServeConfig()
	if err := parseServeFlags(&cfg, nil); err != nil {
		t.Fatal(err)
	}
	if !cfg.Demo {
		t.Fatal("Demo want true from env")
	}
}

func TestCollectFeatures(t *testing.T) {
	t.Setenv("GGHSTATS_METRICS", "false")
	t.Setenv("GGHSTATS_METRICS_PER_REPO", "true")
	t.Setenv("GGHSTATS_CUSTOM_CSS", "/tmp/no-such-gghstats.css")
	t.Setenv("GGHSTATS_RATE_LIMIT_ENABLED", "false")

	f := collectFeatures(serveConfig{
		BadgePublic:   true,
		SyncOnStartup: true,
		APIToken:      "tok",
		PublicURL:     "https://stats.example.com",
		Port:          "9090",
		Host:          "0.0.0.0",
	})
	if !f.BadgePublic || f.MetricsEnabled || !f.MetricsPerRepo {
		t.Fatalf("flags: %+v", f)
	}
	if !f.HasAPIToken || !f.HasPublicURL || !f.HasCustomCSS {
		t.Fatalf("has-*: %+v", f)
	}
	if f.RateLimitEnabled {
		t.Fatal("RateLimitEnabled want false")
	}
	if f.Port != "9090" || f.Host != "0.0.0.0" {
		t.Fatalf("port/host: %+v", f)
	}
}

func TestSetupRateLimiter(t *testing.T) {
	t.Setenv("GGHSTATS_RATE_LIMIT_ENABLED", "false")
	if rl := setupRateLimiter(); rl != nil {
		t.Fatal("want nil when disabled")
	}
	t.Setenv("GGHSTATS_RATE_LIMIT_ENABLED", "true")
	t.Setenv("GGHSTATS_RATE_LIMIT_REQUESTS", "10")
	t.Setenv("GGHSTATS_RATE_LIMIT_PERIOD", "1m")
	t.Setenv("GGHSTATS_RATE_LIMIT_BURST", "2")
	rl := setupRateLimiter()
	if rl == nil {
		t.Fatal("want rate limiter when enabled")
	}
}

func TestResolveCSSPath(t *testing.T) {
	t.Setenv("GGHSTATS_CUSTOM_CSS", "")
	abs, q := resolveCSSPath()
	if abs != "" || q != "" {
		t.Fatalf("empty env: abs=%q q=%q", abs, q)
	}
	t.Setenv("GGHSTATS_CUSTOM_CSS", "/tmp/definitely-missing-gghstats-theme.css")
	abs, q = resolveCSSPath()
	if abs != "" {
		t.Fatalf("missing file should yield empty abs, got %q q=%q", abs, q)
	}
}
