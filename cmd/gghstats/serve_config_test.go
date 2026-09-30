package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/store"
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

func TestIsLoopbackBindHost(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "::1", "[::1]"} {
		if !isLoopbackBindHost(host) {
			t.Errorf("isLoopbackBindHost(%q) = false, want true", host)
		}
	}
	for _, host := range []string{"0.0.0.0", "192.168.1.10", "localhost", ""} {
		if isLoopbackBindHost(host) {
			t.Errorf("isLoopbackBindHost(%q) = true, want false", host)
		}
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

func TestLoadServeConfigUpstreamStaleDefaults(t *testing.T) {
	t.Setenv("GGHSTATS_UPSTREAM_STALE_DAYS", "")
	t.Setenv("GGHSTATS_UPSTREAM_STALE_BANNER", "")
	t.Setenv("GGHSTATS_DEMO_UPSTREAM_STALE", "")
	t.Setenv("GGHSTATS_UPSTREAM_STALE_FORCE", "")
	cfg := loadServeConfig()
	if cfg.UpstreamStaleDays != 3 {
		t.Fatalf("UpstreamStaleDays = %d, want 3", cfg.UpstreamStaleDays)
	}
	if !cfg.UpstreamStaleBanner {
		t.Fatal("UpstreamStaleBanner default want true")
	}
	if cfg.DemoUpstreamStale || cfg.UpstreamStaleForce {
		t.Fatalf("dogfood flags must default off: demo=%v force=%v", cfg.DemoUpstreamStale, cfg.UpstreamStaleForce)
	}
}

func TestLoadServeConfigUpstreamStaleDays(t *testing.T) {
	cases := []struct {
		env  string
		want int
	}{
		{"0", 0},
		{"5", 5},
		{"-1", 3},
		{"nope", 3},
	}
	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			t.Setenv("GGHSTATS_UPSTREAM_STALE_DAYS", tc.env)
			cfg := loadServeConfig()
			if cfg.UpstreamStaleDays != tc.want {
				t.Fatalf("days=%d want %d", cfg.UpstreamStaleDays, tc.want)
			}
		})
	}
}

func TestLoadServeConfigUpstreamStaleBanner(t *testing.T) {
	for _, off := range []string{"false", "0", "off"} {
		t.Run(off, func(t *testing.T) {
			t.Setenv("GGHSTATS_UPSTREAM_STALE_BANNER", off)
			cfg := loadServeConfig()
			if cfg.UpstreamStaleBanner {
				t.Fatalf("BANNER=%q want false", off)
			}
		})
	}
}

func TestLoadServeConfigDemoUpstreamStaleAndForce(t *testing.T) {
	t.Setenv("GGHSTATS_DEMO_UPSTREAM_STALE", "true")
	t.Setenv("GGHSTATS_UPSTREAM_STALE_FORCE", "true")
	cfg := loadServeConfig()
	if !cfg.DemoUpstreamStale || !cfg.UpstreamStaleForce {
		t.Fatalf("%+v", cfg)
	}
}

func TestParseServeFlagsDemoUpstreamStaleAndForce(t *testing.T) {
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
	t.Setenv("GGHSTATS_DEMO", "")
	cfg := loadServeConfig()
	if err := parseServeFlags(&cfg, []string{"--demo", "--demo-upstream-stale", "--upstream-stale-force"}); err != nil {
		t.Fatal(err)
	}
	if !cfg.Demo || !cfg.DemoUpstreamStale || !cfg.UpstreamStaleForce {
		t.Fatalf("%+v", cfg)
	}
}

func TestSeedDemoIfEnabled(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "demo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	if err := seedDemoIfEnabled(db, serveConfig{Demo: false}); err != nil {
		t.Fatal(err)
	}
	n, _ := db.RepoCount()
	if n != 0 {
		t.Fatalf("demo off should not seed, got %d", n)
	}

	if err := seedDemoIfEnabled(db, serveConfig{Demo: true}); err != nil {
		t.Fatal(err)
	}
	n, _ = db.RepoCount()
	if n != 4 {
		t.Fatalf("seed repos=%d", n)
	}

	if err := seedDemoIfEnabled(db, serveConfig{Demo: true, DemoUpstreamStale: true}); err != nil {
		t.Fatal(err)
	}
	got, err := store.DetectFleetUpstreamStale(db, store.ReportVisibility{}, 3, time.Now().UTC(), true)
	if err != nil || !got.Active {
		t.Fatalf("after freeze: %+v err=%v", got, err)
	}

	closed, err := store.Open(filepath.Join(dir, "closed.db"))
	if err != nil {
		t.Fatal(err)
	}
	_ = closed.Close()
	if err := seedDemoIfEnabled(closed, serveConfig{Demo: true}); err == nil {
		t.Fatal("want error seeding closed db")
	}
}
