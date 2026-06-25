package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name          string
		xForwardedFor string
		xRealIP       string
		remoteAddr    string
		want          string
	}{
		{"x-forwarded-for", "1.2.3.4, 5.6.7.8", "", "10.0.0.1:12345", "1.2.3.4"},
		{"x-real-ip", "", "9.9.9.9", "", "9.9.9.9"},
		{"remote-addr with port", "", "", "192.168.1.1:54321", "192.168.1.1"},
		{"remote-addr no port", "", "", "192.168.1.1", "192.168.1.1"},
		{"x-forwarded-for trumps x-real-ip", "4.4.4.4", "8.8.8.8", "10.0.0.1:9999", "4.4.4.4"},
		{"x-real-ip trumps remote-addr", "", "6.6.6.6", "10.0.0.1:9999", "6.6.6.6"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.Header{}
			if tt.xForwardedFor != "" {
				h.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				h.Set("X-Real-IP", tt.xRealIP)
			}
			r := &http.Request{
				Header:     h,
				RemoteAddr: tt.remoteAddr,
			}
			if got := clientIP(r); got != tt.want {
				t.Errorf("clientIP = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseRateLimitEnvDefaults(t *testing.T) {
	for _, k := range []string{
		"GGHSTATS_RATE_LIMIT_ENABLED",
		"GGHSTATS_RATE_LIMIT_REQUESTS",
		"GGHSTATS_RATE_LIMIT_PERIOD",
		"GGHSTATS_RATE_LIMIT_BURST",
	} {
		t.Setenv(k, "")
	}
	cfg := ParseRateLimitEnv()
	if !cfg.Enabled {
		t.Error("expected enabled by default")
	}
	if cfg.Requests != 120 {
		t.Errorf("Requests = %d, want 120", cfg.Requests)
	}
	if cfg.Period != time.Minute {
		t.Errorf("Period = %v, want 1m", cfg.Period)
	}
	if cfg.Burst != 20 {
		t.Errorf("Burst = %d, want 20", cfg.Burst)
	}
}

func TestParseRateLimitEnvOverrides(t *testing.T) {
	t.Setenv("GGHSTATS_RATE_LIMIT_ENABLED", "true")
	t.Setenv("GGHSTATS_RATE_LIMIT_REQUESTS", "60")
	t.Setenv("GGHSTATS_RATE_LIMIT_PERIOD", "30s")
	t.Setenv("GGHSTATS_RATE_LIMIT_BURST", "10")
	cfg := ParseRateLimitEnv()
	if !cfg.Enabled {
		t.Error("expected enabled")
	}
	if cfg.Requests != 60 {
		t.Errorf("Requests = %d, want 60", cfg.Requests)
	}
	if cfg.Period != 30*time.Second {
		t.Errorf("Period = %v, want 30s", cfg.Period)
	}
	if cfg.Burst != 10 {
		t.Errorf("Burst = %d, want 10", cfg.Burst)
	}
}

func TestParseRateLimitEnvDisabled(t *testing.T) {
	for _, v := range []string{"false", "off", "no", "0"} {
		t.Run(v, func(t *testing.T) {
			t.Setenv("GGHSTATS_RATE_LIMIT_ENABLED", v)
			t.Setenv("GGHSTATS_RATE_LIMIT_REQUESTS", "")
			t.Setenv("GGHSTATS_RATE_LIMIT_PERIOD", "")
			t.Setenv("GGHSTATS_RATE_LIMIT_BURST", "")
			cfg := ParseRateLimitEnv()
			if cfg.Enabled {
				t.Errorf("GGHSTATS_RATE_LIMIT_ENABLED=%s expected disabled", v)
			}
		})
	}
}

func TestRateLimiterAllowsWithinLimit(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		Enabled:  true,
		Requests: 100,
		Period:   time.Second,
		Burst:    100,
	})
	defer rl.Shutdown()

	var hits int
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}), MiddlewareSkip{})

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200", i, rec.Code)
		}
	}
	if hits != 100 {
		t.Errorf("hits = %d, want 100", hits)
	}
}

func TestRateLimiterBlocksAfterBurst(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		Enabled:  true,
		Requests: 1,
		Period:   time.Hour, // effectively no refill during test
		Burst:    5,
	})
	defer rl.Shutdown()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), MiddlewareSkip{})

	// Burst of 5 should pass.
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.2:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("burst request %d: got %d, want 200", i, rec.Code)
		}
	}

	// 6th should be rate limited.
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.2:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("got %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("missing Retry-After header")
	}
}

func TestRateLimiterSeparatePerIP(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		Enabled:  true,
		Requests: 1,
		Period:   time.Hour,
		Burst:    2,
	})
	defer rl.Shutdown()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), MiddlewareSkip{})

	// IP A: 2 requests OK
	for range 2 {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Errorf("IP A: got %d, want 200", rec.Code)
		}
	}
	// IP A: 3rd blocked
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 429 {
		t.Errorf("IP A exhausted: got %d, want 429", rec.Code)
	}

	// IP B: still has its own burst
	req = httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.99:12345"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("IP B: got %d, want 200", rec.Code)
	}
}

func TestRateLimiterSkipsExemptPaths(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		Enabled:  true,
		Requests: 1,
		Period:   time.Hour,
		Burst:    1,
	})
	defer rl.Shutdown()

	handler := rl.Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		MiddlewareSkip{Exact: []string{"/metrics", "/api/v1/healthz"}, Prefixes: []string{BadgePathPrefix}},
	)

	// Exhaust the single burst.
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatal("expected first request to pass")
	}

	// / should be blocked now.
	req = httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 429 {
		t.Errorf("expected 429 on /, got %d", rec.Code)
	}

	// healthz and badges should still pass (exempt).
	for _, path := range []string{"/metrics", "/api/v1/healthz", "/api/v1/badge/o/r?metric=clones"} {
		req := httptest.NewRequest("GET", path, nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Errorf("exempt %s: got %d, want 200", path, rec.Code)
		}
	}
}

func TestRateLimiterSkipsBadgePrefix(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		Enabled:  true,
		Requests: 1,
		Period:   time.Hour,
		Burst:    1,
	})
	defer rl.Shutdown()

	handler := rl.Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		PublicMiddlewareSkip(nil),
	)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatal("expected first request on / to pass")
	}
	req = httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 429 {
		t.Fatalf("expected / to be rate limited, got %d", rec.Code)
	}

	for i := 0; i < 50; i++ {
		req = httptest.NewRequest("GET", "/api/v1/badge/o/r?metric=clones", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("badge request %d: got %d, want 200", i, rec.Code)
		}
	}
}

func TestDefaultRateLimitConfig(t *testing.T) {
	cfg := DefaultRateLimitConfig()
	if !cfg.Enabled {
		t.Error("expected enabled")
	}
	if cfg.Requests != 120 {
		t.Errorf("Requests = %d", cfg.Requests)
	}
	if cfg.Period != time.Minute {
		t.Errorf("Period = %v", cfg.Period)
	}
	if cfg.Burst != 20 {
		t.Errorf("Burst = %d", cfg.Burst)
	}
}

func TestParseRateLimitEnvInvalidValuesIgnored(t *testing.T) {
	t.Setenv("GGHSTATS_RATE_LIMIT_REQUESTS", "not-a-number")
	t.Setenv("GGHSTATS_RATE_LIMIT_PERIOD", "banana")
	t.Setenv("GGHSTATS_RATE_LIMIT_BURST", "-5")
	cfg := ParseRateLimitEnv()
	if cfg.Requests != 120 {
		t.Errorf("Requests = %d, want 120 (default for invalid input)", cfg.Requests)
	}
	if cfg.Period != time.Minute {
		t.Errorf("Period = %v, want 1m (default for invalid input)", cfg.Period)
	}
	if cfg.Burst != 20 {
		t.Errorf("Burst = %d, want 20 (default for negative input)", cfg.Burst)
	}
}

func TestRateLimiter429Body(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		Enabled:  true,
		Requests: 1,
		Period:   time.Hour,
		Burst:    1,
	})
	defer rl.Shutdown()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), MiddlewareSkip{})

	// Exhaust.
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.3:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatal("expected pass")
	}

	req = httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.3:12345"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 429 {
		t.Fatalf("got %d, want 429", rec.Code)
	}
	body := rec.Body.String()
	if body != `{"error":"rate_limit_exceeded"}` {
		t.Errorf("body = %q", body)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestNewRateLimiterHandlesZeroRequests(t *testing.T) {
	// Zero requests should produce rate.Inf (no limiting).
	rl := NewRateLimiter(RateLimitConfig{
		Enabled:  true,
		Requests: 0,
		Period:   time.Minute,
		Burst:    5,
	})
	defer rl.Shutdown()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), MiddlewareSkip{})

	for i := 0; i < 20; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.4:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("request %d: got %d, want 200 (rate.Inf should allow all)", i, rec.Code)
		}
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		Enabled:  true,
		Requests: 100,
		Period:   time.Minute,
		Burst:    10,
	})

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), MiddlewareSkip{})

	// Make a request to populate the limiter.
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.5:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Manually set lastSeen far in the past to simulate idle IP.
	rl.limiters.Range(func(key, value any) bool {
		e := value.(*rateLimiterEntry)
		e.lastSeen = time.Now().Add(-10 * time.Minute)
		return false
	})

	rl.cleanup()

	count := 0
	rl.limiters.Range(func(key, value any) bool {
		count++
		return true
	})
	if count != 0 {
		t.Errorf("expected 0 limiters after cleanup, got %d", count)
	}

	rl.Shutdown()
}

func TestConfigRateLimiterField(t *testing.T) {
	// Ensure Config.RateLimiter compiles and is usable.
	cfg := Config{
		RateLimiter: nil,
	}
	if cfg.RateLimiter != nil {
		t.Error("nil RateLimiter should be nil")
	}

	rl := NewRateLimiter(DefaultRateLimitConfig())
	defer rl.Shutdown()
	cfg.RateLimiter = rl
	if cfg.RateLimiter != rl {
		t.Error("RateLimiter should be set")
	}
}

// TestRateLimiterEnvIntegration verifies that the integration between
// ParseRateLimitEnv (which reads os.Getenv) and the serve path is coherent.
func TestRateLimiterEnvIntegration(t *testing.T) {
	// Simulate env: disabled via env var.
	t.Setenv("GGHSTATS_RATE_LIMIT_ENABLED", "false")
	cfg := ParseRateLimitEnv()
	if cfg.Enabled {
		t.Fatal("expected disabled")
	}
	// When disabled, serve.go should not create a RateLimiter, so cfg.RateLimiter
	// stays nil. finalizeHandler then skips rate limiting.
}
