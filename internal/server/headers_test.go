package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	h := securityHeadersMiddleware(Config{}, inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
	if got := w.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q", got)
	}
	if got := w.Header().Get("Referrer-Policy"); got == "" {
		t.Error("missing Referrer-Policy")
	}
	if got := w.Header().Get("Content-Security-Policy-Report-Only"); got == "" {
		t.Error("missing CSP Report-Only")
	}
	if got := w.Header().Get("Content-Security-Policy"); got != "" {
		t.Errorf("unexpected enforce CSP = %q", got)
	}
}

func TestSecurityHeadersCSPEnforce(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	h := securityHeadersMiddleware(Config{CSPMode: "enforce"}, inner)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("expected Content-Security-Policy")
	}
	if w.Header().Get("Content-Security-Policy-Report-Only") != "" {
		t.Fatal("did not expect Report-Only when enforce")
	}
}

func TestSecurityHeadersCSPEnforceWithHeadHTML(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	h := securityHeadersMiddleware(Config{CSPMode: "enforce", HeadHTML: "<script></script>"}, inner)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Header().Get("Content-Security-Policy-Report-Only") == "" {
		t.Fatal("expected Report-Only when HeadHTML set")
	}
	if w.Header().Get("Content-Security-Policy") != "" {
		t.Fatal("enforce must not apply with HeadHTML")
	}
}

func TestLogMiddlewareRecordsStatus(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	h := logMiddleware(nil, inner)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusTeapot {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestLogMiddlewareLogsClientIPFromXFF(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	trusted := ParseTrustedProxies("10.0.0.0/8")
	h := logMiddleware(trusted, inner)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "10.1.2.3:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.50")
	h.ServeHTTP(httptest.NewRecorder(), req)

	out := buf.String()
	if !strings.Contains(out, `"ip":"203.0.113.50"`) {
		t.Fatalf("access log missing client ip from XFF: %s", out)
	}
}

func TestLogMiddlewareLogsPeerIPWithoutTrusted(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := logMiddleware(nil, inner)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "10.1.2.3:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.50")
	h.ServeHTTP(httptest.NewRecorder(), req)

	out := buf.String()
	if !strings.Contains(out, `"ip":"10.1.2.3"`) {
		t.Fatalf("access log should use peer when untrusted: %s", out)
	}
	if strings.Contains(out, `"ip":"203.0.113.50"`) {
		t.Fatalf("must not trust XFF without trusted proxies: %s", out)
	}
}

func TestHttpAccessLogLevel(t *testing.T) {
	cases := []struct {
		status int
		want   slog.Level
	}{
		{200, slog.LevelInfo},
		{301, slog.LevelInfo},
		{404, slog.LevelWarn},
		{418, slog.LevelWarn},
		{500, slog.LevelError},
		{503, slog.LevelError},
	}
	for _, tc := range cases {
		if got := httpAccessLogLevel(tc.status); got != tc.want {
			t.Errorf("httpAccessLogLevel(%d) = %v, want %v", tc.status, got, tc.want)
		}
	}
}
