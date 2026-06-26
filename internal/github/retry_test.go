package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetrySuccessFirstAttempt(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("tok")
	c.BaseURL = srv.URL
	c.SetRetry(RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond})

	var out map[string]any
	if err := c.getCtx(context.Background(), "/x", "application/vnd.github+json", &out); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestRetryExhaustsOnServerError(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient("tok")
	c.BaseURL = srv.URL
	c.SetRetry(RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond})

	var out map[string]string
	err := c.getCtx(context.Background(), "/x", "application/vnd.github+json", &out)
	if err == nil {
		t.Fatal("expected error after retries")
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
}

func TestRetryRecoversAfterTransient(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewClient("tok")
	c.BaseURL = srv.URL
	c.SetRetry(RetryConfig{MaxAttempts: 4, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond})

	var out map[string]bool
	if err := c.getCtx(context.Background(), "/x", "application/vnd.github+json", &out); err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
}

func TestRetryDoesNotRetry4xx(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient("tok")
	c.BaseURL = srv.URL
	c.SetRetry(RetryConfig{MaxAttempts: 4, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond})

	var out map[string]string
	if err := c.getCtx(context.Background(), "/x", "application/vnd.github+json", &out); err == nil {
		t.Fatal("expected 404 error")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1 (no retry on plain 4xx)", calls)
	}
}

func TestRetryRespects429AndRateReset(t *testing.T) {
	var calls atomic.Int32
	// Round up to next whole second: Unix() truncates fractional seconds,
	// so we add enough slack to guarantee time.Until(reset) > 0 when the
	// sleep actually runs.
	reset := strconv.FormatInt(time.Now().Add(2*time.Second).Unix(), 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", reset)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("tok")
	c.BaseURL = srv.URL
	// MaxDelay >> reset (~2s): ensures backoffDelay picks the reset hint
	// instead of the exp-backoff cap.
	c.SetRetry(RetryConfig{MaxAttempts: 3, BaseDelay: 10 * time.Millisecond, MaxDelay: 10 * time.Second})

	start := time.Now()
	var out map[string]any
	if err := c.getCtx(context.Background(), "/x", "application/vnd.github+json", &out); err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}
	// Reset hint (~2s) should dominate: elapsed must be at least ~1s. Use a
	// floor below the truncated reset to avoid CI flake without being so
	// lax the test is meaningless.
	if d := time.Since(start); d < 800*time.Millisecond {
		t.Fatalf("did not honor reset wait, elapsed = %v", d)
	}
}

func TestShouldRetryMatrix(t *testing.T) {
	mkResp := func(code int, remaining string) *http.Response {
		r := &http.Response{Header: http.Header{}}
		r.StatusCode = code
		if remaining != "" {
			r.Header.Set("X-RateLimit-Remaining", remaining)
		}
		return r
	}
	cases := []struct {
		name string
		resp *http.Response
		err  error
		want bool
	}{
		{"500", mkResp(500, ""), nil, true},
		{"502", mkResp(502, ""), nil, true},
		{"429", mkResp(429, ""), nil, true},
		{"403-rate", mkResp(403, "0"), nil, true},
		{"403-not-rate", mkResp(403, "5"), nil, false},
		{"404", mkResp(404, ""), nil, false},
		{"400", mkResp(400, ""), nil, false},
		{"200", mkResp(200, ""), nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldRetry(tc.resp, tc.err)
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBackoffDelayHonorsReset(t *testing.T) {
	cfg := RetryConfig{MaxAttempts: 3, BaseDelay: time.Second, MaxDelay: 10 * time.Second}
	r := &http.Response{Header: http.Header{}}
	// Reset must be sufficiently in the future that Unix() truncation and
	// the gap between header parse and backoffDelay call cannot push it into
	// the past. 5s of slack is more than enough.
	r.Header.Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(5*time.Second).Unix(), 10))
	d := backoffDelay(cfg, 0, r)
	if d <= 0 {
		t.Fatalf("expected positive reset wait, got %v", d)
	}
	if d > 6*time.Second {
		t.Fatalf("reset wait too long, got %v", d)
	}
}

func TestBackoffDelayExpJitter(t *testing.T) {
	cfg := RetryConfig{MaxAttempts: 5, BaseDelay: 100 * time.Millisecond, MaxDelay: 5 * time.Second}
	for attempt := 0; attempt < 3; attempt++ {
		d := backoffDelay(cfg, attempt, nil)
		max := cfg.BaseDelay << attempt
		if max > cfg.MaxDelay {
			max = cfg.MaxDelay
		}
		if d < 0 || d > max {
			t.Fatalf("attempt %d: delay %v outside [0,%v]", attempt, d, max)
		}
	}
}
