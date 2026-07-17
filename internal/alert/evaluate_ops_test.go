package alert

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/store"
)

func TestRunOpsRules_RepoFetchFailed(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ApplyRetryConfig(RetryConfig{MaxAttempts: 1, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond})
	t.Cleanup(func() { ApplyRetryConfig(DefaultRetryConfig) })

	cfg := EvalConfig{
		DB: db,
		Rules: []RuleSpec{{
			Kind: KindOps, Event: "repo_fetch_failed", Window: "this_sync",
			Op: "gte", Value: 3, Level: "warn", Debounce: "every_sync",
		}},
		Senders: BuildSenders([]ResolvedSink{{Type: TypeSlack, URL: srv.URL}}, srv.Client()),
		Now:     time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC),
	}
	RunOpsRules(context.Background(), cfg, SyncSnapshot{
		Success: true, ReposAttempted: 10, ReposFailed: 4,
		FailedRepos: []string{"a/b", "c/d"}, RateLimitRemaining: 5000,
	})
	if n != 1 {
		t.Fatalf("want 1 alert, got %d", n)
	}
	// below threshold
	RunOpsRules(context.Background(), cfg, SyncSnapshot{
		Success: true, ReposAttempted: 10, ReposFailed: 1, RateLimitRemaining: 5000,
	})
	if n != 1 {
		t.Fatalf("want still 1, got %d", n)
	}
}

func TestRunOpsRules_SyncFailedConsecutive(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ApplyRetryConfig(RetryConfig{MaxAttempts: 1, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond})
	t.Cleanup(func() { ApplyRetryConfig(DefaultRetryConfig) })

	cfg := EvalConfig{
		DB: db,
		Rules: []RuleSpec{{
			Kind: KindOps, Event: "sync_failed", Window: "consecutive_runs",
			Op: "gte", Value: 2, Level: "crit",
		}},
		Senders: BuildSenders([]ResolvedSink{{Type: TypeSlack, URL: srv.URL}}, srv.Client()),
		Now:     time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC),
	}
	RunOpsRules(context.Background(), cfg, SyncSnapshot{Success: false, RateLimitRemaining: -1})
	if n != 0 {
		t.Fatalf("first failure should not fire (need 2), got %d", n)
	}
	RunOpsRules(context.Background(), cfg, SyncSnapshot{Success: false, RateLimitRemaining: -1})
	if n != 1 {
		t.Fatalf("second consecutive failure should fire, got %d", n)
	}
}

func TestRunOpsRules_RateLimit(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ApplyRetryConfig(RetryConfig{MaxAttempts: 1, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond})
	t.Cleanup(func() { ApplyRetryConfig(DefaultRetryConfig) })

	cfg := EvalConfig{
		DB: db,
		Rules: []RuleSpec{{
			Kind: KindOps, Event: "rate_limit", Op: "lt", Value: 100, Level: "warn", Debounce: "every_sync",
		}},
		Senders: BuildSenders([]ResolvedSink{{Type: TypeSlack, URL: srv.URL}}, srv.Client()),
		Now:     time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC),
	}
	RunOpsRules(context.Background(), cfg, SyncSnapshot{Success: true, RateLimitRemaining: 87})
	if n != 1 {
		t.Fatalf("want 1, got %d", n)
	}
}
