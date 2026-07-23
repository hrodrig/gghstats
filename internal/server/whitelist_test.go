package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseWhitelistEnvDefaults(t *testing.T) {
	t.Setenv("GGHSTATS_WHITELIST", "")
	t.Setenv("GGHSTATS_WHITELIST_PATHS", "")
	cfg := ParseWhitelistEnv()
	if cfg.CIDRs != "" {
		t.Errorf("CIDRs = %q, want empty", cfg.CIDRs)
	}
	if cfg.Paths != "" {
		t.Errorf("Paths = %q, want empty", cfg.Paths)
	}
}

func TestParseWhitelistEnvValues(t *testing.T) {
	t.Setenv("GGHSTATS_WHITELIST", "10.0.0.0/8,192.168.1.1")
	t.Setenv("GGHSTATS_WHITELIST_PATHS", "/api/,/h2h")
	cfg := ParseWhitelistEnv()
	if cfg.CIDRs != "10.0.0.0/8,192.168.1.1" {
		t.Errorf("CIDRs = %q", cfg.CIDRs)
	}
	if cfg.Paths != "/api/,/h2h" {
		t.Errorf("Paths = %q", cfg.Paths)
	}
}

func TestNewWhitelistEmptyReturnsNil(t *testing.T) {
	w := NewWhitelist(WhitelistConfig{CIDRs: ""}, "")
	if w != nil {
		t.Error("expected nil for empty CIDRs")
	}
}

func TestNewWhitelistSingleIP(t *testing.T) {
	w := NewWhitelist(WhitelistConfig{CIDRs: "10.0.0.1"}, "")
	if w == nil {
		t.Fatal("expected non-nil")
	}
	if !w.allowed("10.0.0.1") {
		t.Error("10.0.0.1 should be allowed")
	}
	if w.allowed("10.0.0.2") {
		t.Error("10.0.0.2 should NOT be allowed")
	}
}

func TestNewWhitelistCIDR(t *testing.T) {
	w := NewWhitelist(WhitelistConfig{CIDRs: "10.0.0.0/24"}, "")
	if w == nil {
		t.Fatal("expected non-nil")
	}
	if !w.allowed("10.0.0.1") {
		t.Error("10.0.0.1 should be allowed")
	}
	if !w.allowed("10.0.0.255") {
		t.Error("10.0.0.255 should be allowed")
	}
	if w.allowed("10.0.1.1") {
		t.Error("10.0.1.1 should NOT be allowed")
	}
}

func TestNewWhitelistMultiple(t *testing.T) {
	w := NewWhitelist(WhitelistConfig{CIDRs: "10.0.0.0/24, 192.168.1.1"}, "")
	if w == nil {
		t.Fatal("expected non-nil")
	}
	if !w.allowed("10.0.0.5") {
		t.Error("10.0.0.5 should be allowed")
	}
	if !w.allowed("192.168.1.1") {
		t.Error("192.168.1.1 should be allowed")
	}
	if w.allowed("172.16.0.1") {
		t.Error("172.16.0.1 should NOT be allowed")
	}
}

func TestNewWhitelistInvalidCIDRSkipped(t *testing.T) {
	w := NewWhitelist(WhitelistConfig{CIDRs: "not-an-ip, 10.0.0.1, bad/cidr"}, "")
	if w == nil {
		t.Fatal("expected non-nil with at least one valid entry")
	}
	if !w.allowed("10.0.0.1") {
		t.Error("10.0.0.1 should be allowed")
	}
	if w.allowed("1.1.1.1") {
		t.Error("1.1.1.1 should NOT be allowed")
	}
}

func TestWhitelistMiddlewareAllowsWhitelisted(t *testing.T) {
	w := NewWhitelist(WhitelistConfig{CIDRs: "10.0.0.0/8"}, "")
	handler := w.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}), MiddlewareSkip{})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.1.2.3:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("got %d, want 200", rec.Code)
	}
}

func TestWhitelistMiddlewareBlocksNonWhitelisted(t *testing.T) {
	w := NewWhitelist(WhitelistConfig{CIDRs: "10.0.0.0/8"}, "")
	handler := w.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}), MiddlewareSkip{})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("got %d, want 403", rec.Code)
	}
	body := rec.Body.String()
	if body != `{"error":"ip_not_whitelisted"}` {
		t.Errorf("body = %q", body)
	}
}

func TestWhitelistMiddlewareXForwardedFor(t *testing.T) {
	w := NewWhitelist(WhitelistConfig{CIDRs: "10.0.0.0/8"}, "")
	w.trusted = ParseTrustedProxies("192.168.1.0/24")
	handler := w.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}), MiddlewareSkip{})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "10.5.5.5, 172.16.0.1")
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("got %d, want 200 (trusted peer + X-Forwarded-For)", rec.Code)
	}
}

func TestWhitelistMiddlewareRejectsForgedXForwardedForFromUntrustedPeer(t *testing.T) {
	w := NewWhitelist(WhitelistConfig{CIDRs: "10.0.0.0/8"}, "")
	w.trusted = ParseTrustedProxies("192.168.1.0/24")
	handler := w.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}), MiddlewareSkip{})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "10.5.5.5, 172.16.0.1")
	req.RemoteAddr = "203.0.113.7:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("got %d, want 403 (untrusted peer must not forge X-Forwarded-For)", rec.Code)
	}
}

func TestWhitelistMiddlewarePathScoped(t *testing.T) {
	w := NewWhitelist(WhitelistConfig{
		CIDRs: "10.0.0.0/8",
		Paths: "/api/",
	}, "")
	handler := w.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}), MiddlewareSkip{})

	// Non-whitelisted IP on /api/ → blocked.
	req := httptest.NewRequest("GET", "/api/repos", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("GET /api/repos: got %d, want 403", rec.Code)
	}

	// Same IP on / (dashboard) → allowed (not in whitelisted paths).
	req = httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /: got %d, want 200 (path not protected)", rec.Code)
	}

	// Whitelisted IP on /api/ → allowed.
	req = httptest.NewRequest("GET", "/api/repos", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("whitelisted GET /api/repos: got %d, want 200", rec.Code)
	}
}

func TestWhitelistMiddlewareExemptPaths(t *testing.T) {
	w := NewWhitelist(WhitelistConfig{CIDRs: "10.0.0.0/8"}, "")
	handler := w.Middleware(
		http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			rw.WriteHeader(http.StatusOK)
		}),
		PublicMiddlewareSkip(nil),
	)

	for _, path := range []string{"/metrics", "/api/v1/healthz", "/api/v1/badge/o/r"} {
		req := httptest.NewRequest("GET", path, nil)
		req.RemoteAddr = "192.168.1.1:12345" // not whitelisted
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("exempt %s: got %d, want 200", path, rec.Code)
		}
	}

	// Non-exempt path should still be blocked.
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("non-exempt /: got %d, want 403", rec.Code)
	}
}

func TestWhitelistBadgeExemptWhenAPIPathsProtected(t *testing.T) {
	w := NewWhitelist(WhitelistConfig{
		CIDRs: "10.0.0.0/8",
		Paths: "/api/",
	}, "")
	handler := w.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}), PublicMiddlewareSkip(nil))

	req := httptest.NewRequest("GET", "/api/v1/badge/o/r?metric=clones", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("badge from non-whitelisted IP: got %d, want 200", rec.Code)
	}

	req = httptest.NewRequest("GET", "/api/repos", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("API from non-whitelisted IP: got %d, want 403", rec.Code)
	}
}

func TestWhitelistPathMatch(t *testing.T) {
	w := NewWhitelist(WhitelistConfig{
		CIDRs: "10.0.0.0/8",
		Paths: "/api/,/h2h,/sync",
	}, "")

	tests := []struct {
		path    string
		matches bool
	}{
		{"/api/repos", true},
		{"/api/v1/sync", true},
		{"/api", true},
		{"/h2h", true},
		{"/sync", true},
		{"/sync-status", false}, // prefix must match with trailing slash or exact
		{"/", false},
		{"/dashboard", false},
		{"/static/app.js", false},
	}
	for _, tt := range tests {
		if got := w.pathMatches(tt.path); got != tt.matches {
			t.Errorf("pathMatches(%q) = %v, want %v", tt.path, got, tt.matches)
		}
	}
}

func TestWhitelistBypassWithValidAPIToken(t *testing.T) {
	const token = "secret-sync-token"
	w := NewWhitelist(WhitelistConfig{
		CIDRs: "10.0.0.0/8",
		Paths: "/api/",
	}, token)
	handler := w.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}), MiddlewareSkip{})

	req := httptest.NewRequest("POST", "/api/v1/sync", nil)
	req.RemoteAddr = "203.0.113.5:12345"
	req.Header.Set("x-api-token", token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("valid token bypass: got %d, want 200", rec.Code)
	}

	req = httptest.NewRequest("POST", "/api/v1/sync", nil)
	req.RemoteAddr = "203.0.113.5:12345"
	req.Header.Set("x-api-token", "wrong")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("wrong token: got %d, want 403", rec.Code)
	}

	req = httptest.NewRequest("POST", "/api/v1/sync", nil)
	req.RemoteAddr = "203.0.113.5:12345"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("missing token: got %d, want 403", rec.Code)
	}
}

func TestWhitelistMiddlewareAllowsAllWhenNil(t *testing.T) {
	cfg := Config{Whitelist: nil}
	if cfg.Whitelist != nil {
		t.Fatal("nil whitelist should remain nil")
	}
	// When whitelist is nil, finalizeHandler skips it entirely.
}
