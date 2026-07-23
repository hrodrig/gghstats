package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
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
	h := logMiddleware(inner)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusTeapot {
		t.Fatalf("status = %d", w.Code)
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
