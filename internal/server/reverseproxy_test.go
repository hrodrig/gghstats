package server

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseReverseProxyRules(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int // number of rules expected
	}{
		{"empty", "", 0},
		{"invalid json", "not-json", 0},
		{"empty array", "[]", 0},
		{"single rule", `[{"local":"/kiko","url":"https://events.example.com"}]`, 1},
		{"rule with headers", `[{"local":"/kiko","url":"https://events.example.com","headers":{"Host":"kiko-backend"}}]`, 1},
		{"multiple rules", `[{"local":"/kiko","url":"https://a.com"},{"local":"/pepe","url":"https://b.com"}]`, 2},
		{"rule with empty local", `[{"local":"","url":"https://a.com"}]`, 1}, // parsed but filtered at mount
		{"rule with empty url", `[{"local":"/kiko","url":""}]`, 1},           // parsed but filtered at mount
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseReverseProxyRules(tt.raw)
			if len(got) != tt.want {
				t.Errorf("got %d rules, want %d (rules: %+v)", len(got), tt.want, got)
			}
		})
	}
}

func TestReverseProxyHandler(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "method=%s path=%s host=%s custom=%s", r.Method, r.URL.Path, r.Host, r.Header.Get("X-Custom"))
	}))
	defer backend.Close()

	rule := ReverseProxyRule{
		Local: "/kiko",
		URL:   backend.URL,
		Headers: map[string]string{
			"Host":     "kiko-backend",
			"X-Custom": "abc123",
		},
	}
	handler := newReverseProxyHandler(rule)

	tests := []struct {
		name   string
		method string
		path   string
		want   string
	}{
		{
			name:   "GET kiko.js",
			method: "GET",
			path:   "/kiko/kiko.js",
			want:   "method=GET path=/kiko.js host=kiko-backend custom=abc123",
		},
		{
			name:   "POST api",
			method: "POST",
			path:   "/kiko/api",
			want:   "method=POST path=/api host=kiko-backend custom=abc123",
		},
		{
			name:   "GET api.gif",
			method: "GET",
			path:   "/kiko/api.gif",
			want:   "method=GET path=/api.gif host=kiko-backend custom=abc123",
		},
		{
			name:   "root",
			method: "GET",
			path:   "/kiko/",
			want:   "method=GET path=/ host=kiko-backend custom=abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			got := strings.TrimSpace(string(body))
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReverseProxyHandlerModifyResponse(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	rule := ReverseProxyRule{
		Local: "/kiko",
		URL:   backend.URL,
	}
	handler := newReverseProxyHandler(rule)

	req := httptest.NewRequest("GET", "/kiko/hello", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	resp.Body.Close()

	if v := resp.Header.Get("Content-Security-Policy"); v != "" {
		t.Errorf("Content-Security-Policy header should be stripped, got %q", v)
	}
}

func TestReverseProxyIntegrationInServer(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "path=%s host=%s", r.URL.Path, r.Host)
	}))
	defer backend.Close()

	db := testStore(t)
	handler := New(Config{
		Store: db,
		ReverseProxyRules: []ReverseProxyRule{
			{
				Local: "/kiko",
				URL:   backend.URL,
				Headers: map[string]string{
					"Host": "kiko-backend",
				},
			},
		},
	})

	req := httptest.NewRequest("GET", "/kiko/kiko.js", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	w.Result().Body.Close()

	got := strings.TrimSpace(string(body))
	want := "path=/kiko.js host=kiko-backend"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestHeadHTMLInLayout(t *testing.T) {
	db := testStore(t)
	handler := New(Config{
		Store:    db,
		HeadHTML: template.HTML(`<script defer src="https://example.com/test.js"></script><link rel="stylesheet" href="/test.css">`),
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()

	if !strings.Contains(body, `src="https://example.com/test.js"`) {
		t.Error("layout does not contain the injected script")
	}
	if !strings.Contains(body, `href="/test.css"`) {
		t.Error("layout does not contain the injected stylesheet link")
	}
}

func TestPublicMiddlewareSkipDerivesProxyPrefixes(t *testing.T) {
	rules := []ReverseProxyRule{
		{Local: "/kiko", URL: "https://events.example.com"},
		{Local: "/custom-proxy", URL: "https://other.example.com"},
		{Local: "", URL: "https://empty.example.com"}, // should be skipped
	}
	skip := PublicMiddlewareSkip(rules)

	if !skip.matches("/kiko/kiko.js") {
		t.Error("/kiko/kiko.js should match (exempt)")
	}
	if !skip.matches("/custom-proxy/api") {
		t.Error("/custom-proxy/api should match (exempt)")
	}
	if !skip.matches("/static/app.js") {
		t.Error("/static/app.js should match (exempt)")
	}
	if !skip.matches("/api/v1/badge/o/r") {
		t.Error("/api/v1/badge/o/r should match (exempt)")
	}
	if skip.matches("/api/repos") {
		t.Error("/api/repos should NOT match (not exempt)")
	}
	if skip.matches("/") {
		t.Error("/ should NOT match (not exempt)")
	}
}

func TestReverseProxyCustomLocalExemptFromMiddleware(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "path=%s", r.URL.Path)
	}))
	defer backend.Close()

	db := testStore(t)
	handler := New(Config{
		Store: db,
		ReverseProxyRules: []ReverseProxyRule{
			{
				Local: "/analytics",
				URL:   backend.URL,
			},
		},
		// Enable rate limiter with tight limit so non-exempt paths get blocked.
		RateLimiter: NewRateLimiter(RateLimitConfig{
			Enabled:  true,
			Requests: 1,
			Period:   time.Hour, // no refill during test
			Burst:    1,
		}),
		Whitelist: NewWhitelist(WhitelistConfig{CIDRs: "10.0.0.0/8"}, ""),
	})

	// Exhaust burst on a non-exempt path (from a non-whitelisted IP).
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests && w.Code != http.StatusForbidden {
		t.Fatalf("expected rate-limit or whitelist block on /, got %d", w.Code)
	}

	// The custom proxy path /analytics should still be reachable (exempt).
	req = httptest.NewRequest("GET", "/analytics/script.js", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	w.Result().Body.Close()
	got := strings.TrimSpace(string(body))
	want := "path=/script.js"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestReverseProxyHandlerContentTypeCSS(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("body {}"))
	}))
	defer backend.Close()

	rule := ReverseProxyRule{Local: "/proxy", URL: backend.URL}
	handler := newReverseProxyHandler(rule)

	req := httptest.NewRequest("GET", "/proxy/styles.css", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	resp.Body.Close()

	// The proxy forwards the upstream Content-Type — it should still be text/plain
	// since ModifyResponse only overrides .js. No CSP should be present.
	if v := resp.Header.Get("Content-Security-Policy"); v != "" {
		t.Errorf("CSP should be stripped, got %q", v)
	}
	// Verify the upstream body was forwarded.
	if w.Body.String() != "body {}" {
		t.Errorf("body = %q, want %q", w.Body.String(), "body {}")
	}
}

func TestReverseProxyHandlerContentTypeJS(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("console.log('hi')"))
	}))
	defer backend.Close()

	rule := ReverseProxyRule{Local: "/proxy", URL: backend.URL}
	handler := newReverseProxyHandler(rule)

	req := httptest.NewRequest("GET", "/proxy/app.js", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	resp.Body.Close()

	// .js files should be overridden to application/javascript.
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/javascript") {
		t.Errorf("Content-Type for .js = %q, want application/javascript", ct)
	}
}

func TestReverseProxyHandlerWithGIF(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/gif")
		w.Write([]byte("GIF89a")) // minimal GIF header
	}))
	defer backend.Close()

	rule := ReverseProxyRule{Local: "/proxy", URL: backend.URL}
	handler := newReverseProxyHandler(rule)

	req := httptest.NewRequest("GET", "/proxy/tracker.gif", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/gif") {
		t.Errorf("Content-Type for .gif = %q, want image/gif", ct)
	}
	if !strings.Contains(w.Body.String(), "GIF89a") {
		t.Errorf("body should contain GIF data")
	}
}

func TestReverseProxyHandlerLargeBody(t *testing.T) {
	// Generate 2 MB of data.
	largeBody := make([]byte, 2*1024*1024)
	for i := range largeBody {
		largeBody[i] = byte(i % 256)
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(largeBody)
	}))
	defer backend.Close()

	rule := ReverseProxyRule{Local: "/proxy", URL: backend.URL}
	handler := newReverseProxyHandler(rule)

	req := httptest.NewRequest("GET", "/proxy/data.bin", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("reading large body: %v", err)
	}
	if len(body) != len(largeBody) {
		t.Fatalf("body size = %d, want %d", len(body), len(largeBody))
	}
	for i := range body {
		if body[i] != largeBody[i] {
			t.Fatalf("body mismatch at byte %d", i)
		}
	}
}

func TestReverseProxyHandlerUpstreamTimeout(t *testing.T) {
	// Backend that never responds until the context is cancelled.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer backend.Close()

	rule := ReverseProxyRule{Local: "/proxy", URL: backend.URL}
	handler := newReverseProxyHandler(rule)

	// Use a short timeout context so the request fails fast.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := httptest.NewRequestWithContext(ctx, "GET", "/proxy/hang", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// The proxy should return an error (502) or at least not hang forever.
	resp := w.Result()
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway && resp.StatusCode != http.StatusOK {
		t.Errorf("expected 502 (Bad Gateway) or 200, got %d", resp.StatusCode)
	}
}

// --- logformat tests ---

func TestNewFormatLogHandlerWritesToStderrWhenNil(t *testing.T) {
	// When writer is nil, it should use os.Stderr without panicking.
	h := NewFormatLogHandler(nil, slog.LevelInfo)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if !h.Enabled(nil, slog.LevelInfo) {
		t.Error("expected info level to be enabled")
	}
	if h.Enabled(nil, slog.LevelDebug) {
		t.Error("expected debug level to be disabled")
	}
}

func TestFormatLogHandlerHandleWritesCorrectFormat(t *testing.T) {
	var buf strings.Builder
	h := NewFormatLogHandler(&buf, slog.LevelInfo)

	record := slog.NewRecord(time.Date(2026, 6, 25, 1, 0, 0, 0, time.UTC), slog.LevelInfo, "test message", 0)
	record.Add("key1", "value1", "key2", 42)

	if err := h.Handle(nil, record); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, " - gghstats - INFO - test message") {
		t.Errorf("unexpected format: %q", out)
	}
	if !strings.Contains(out, "key1=value1") {
		t.Errorf("missing attr key1: %q", out)
	}
	if !strings.Contains(out, "key2=42") {
		t.Errorf("missing attr key2: %q", out)
	}
}

func TestFormatLogHandlerHandleNoAttrs(t *testing.T) {
	var buf strings.Builder
	h := NewFormatLogHandler(&buf, slog.LevelInfo)

	record := slog.NewRecord(time.Date(2026, 6, 25, 1, 0, 0, 0, time.UTC), slog.LevelWarn, "no attrs", 0)

	if err := h.Handle(nil, record); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, " - gghstats - WARN - no attrs") {
		t.Errorf("unexpected format: %q", out)
	}
	// Should end with newline, no trailing key=value.
	if !strings.HasSuffix(out, "no attrs\n") {
		t.Errorf("expected line to end with message + newline, got %q", out)
	}
}

func TestFormatLogHandlerWithAttrsAndGroupAreNoops(t *testing.T) {
	var buf strings.Builder
	h := NewFormatLogHandler(&buf, slog.LevelInfo)

	h2 := h.WithAttrs([]slog.Attr{slog.String("ignored", "yes")})
	if h2 != h {
		t.Error("WithAttrs should return same handler (noop)")
	}

	h3 := h.WithGroup("group")
	if h3 != h {
		t.Error("WithGroup should return same handler (noop)")
	}
}

// --- reverse-proxy mount tests ---

func TestMountReverseProxyRoutesSkipsEmptyLocalOrURL(t *testing.T) {
	mux := http.NewServeMux()

	// Rule with empty local should be skipped without panic.
	mountReverseProxyRoutes(mux, []ReverseProxyRule{
		{Local: "", URL: "https://example.com"},
	})
	// Rule with empty URL should be skipped without panic.
	mountReverseProxyRoutes(mux, []ReverseProxyRule{
		{Local: "/test", URL: ""},
	})
	// No routes should be registered (handler would 404).
	req := httptest.NewRequest("GET", "/test/foo", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for skipped rule, got %d", w.Code)
	}
}

func TestMustParseURLPanicsOnInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid URL")
		}
	}()
	mustParseURL("://invalid-url")
}

func TestReverseProxyHandlerWithoutHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "path=%s", r.URL.Path)
	}))
	defer backend.Close()

	rule := ReverseProxyRule{
		Local: "/proxy",
		URL:   backend.URL,
		// Headers is nil — should not create headerOverrideTransport.
	}
	handler := newReverseProxyHandler(rule)

	req := httptest.NewRequest("GET", "/proxy/script.js", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	got := strings.TrimSpace(string(body))
	want := "path=/script.js"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestHeadHTMLIsEmptyByDefault(t *testing.T) {
	db := testStore(t)
	handler := New(Config{
		Store: db,
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()

	// There should be no stray head-html marker in the output.
	if strings.Contains(body, "HeadHTML") {
		t.Error("layout contains unexpected HeadHTML reference")
	}
}
