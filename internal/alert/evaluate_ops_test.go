package alert

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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

func TestDefaultOpsLevel(t *testing.T) {
	if got := defaultOpsLevel("sync_failed"); got != "crit" {
		t.Fatalf("sync_failed=%q", got)
	}
	if got := defaultOpsLevel("github_unreachable"); got != "crit" {
		t.Fatalf("github_unreachable=%q", got)
	}
	if got := defaultOpsLevel("rate_limit"); got != "warn" {
		t.Fatalf("rate_limit=%q", got)
	}
	if got := defaultOpsLevel("upstream_stale"); got != "warn" {
		t.Fatalf("upstream_stale=%q", got)
	}
}

func TestRunOpsRules_UpstreamStaleOncePerEpisode(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var n int
	var lastBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		b, _ := io.ReadAll(r.Body)
		lastBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ApplyRetryConfig(RetryConfig{MaxAttempts: 1, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond})
	t.Cleanup(func() { ApplyRetryConfig(DefaultRetryConfig) })

	rule := RuleSpec{
		Kind: KindOps, Event: "upstream_stale", Op: "gte", Value: 1,
		// Empty Debounce → opsDebounceMode "once" (episode, not once_per_utc_day).
	}
	if got := rule.opsDebounceMode(); got != "once" {
		t.Fatalf("opsDebounceMode = %q, want once", got)
	}
	// identityKey still uses debounceMode() (empty → once_per_utc_day) for the stamp key shape.
	wantKey := rule.identityKey(rule.Event)
	if wantKey != "ops|upstream_stale|||gte|1|once_per_utc_day" {
		t.Fatalf("rule_key shape = %q", wantKey)
	}

	cfg := EvalConfig{
		DB:      db,
		Rules:   []RuleSpec{rule},
		Senders: BuildSenders([]ResolvedSink{{Type: TypeSlack, URL: srv.URL}}, srv.Client()),
		Now:     time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC),
	}

	stuck := SyncSnapshot{
		Success: true, RateLimitRemaining: 5000,
		UpstreamStale: true, UpstreamStaleSince: "2026-09-23",
	}
	RunOpsRules(context.Background(), cfg, stuck)
	if n != 1 {
		t.Fatalf("enter stuck: want 1 delivery, got %d", n)
	}
	if !strings.Contains(lastBody, "upstream_stale") || !strings.Contains(lastBody, "2026-09-23") {
		t.Fatalf("payload missing event/since: %s", lastBody)
	}
	stamp, err := db.AlertDebounceGet(wantKey)
	if err != nil || stamp != "fired" {
		t.Fatalf("debounce stamp after fire: %q err=%v", stamp, err)
	}

	RunOpsRules(context.Background(), cfg, stuck)
	if n != 1 {
		t.Fatalf("still stuck: want no second delivery, got %d", n)
	}

	clear := SyncSnapshot{Success: true, RateLimitRemaining: 5000, UpstreamStale: false}
	RunOpsRules(context.Background(), cfg, clear)
	stamp, err = db.AlertDebounceGet(wantKey)
	if err != nil || stamp != "" {
		t.Fatalf("after recover stamp should clear: %q err=%v", stamp, err)
	}
	if n != 1 {
		t.Fatalf("recover must not deliver: got %d", n)
	}

	RunOpsRules(context.Background(), cfg, stuck)
	if n != 2 {
		t.Fatalf("re-enter stuck: want 2nd delivery, got %d", n)
	}
}

func TestOpsEventMetrics_UpstreamStale(t *testing.T) {
	_, _, _, skip, err := opsEventMetrics("upstream_stale", SyncSnapshot{}, 0, "this_sync")
	if err != nil || !skip {
		t.Fatalf("inactive want skip err=%v skip=%v", err, skip)
	}
	count, detail, win, skip, err := opsEventMetrics("upstream_stale", SyncSnapshot{
		UpstreamStale: true, UpstreamStaleSince: "2026-09-23",
	}, 0, "this_sync")
	if err != nil || skip || count != 1 || win != "this_sync" {
		t.Fatalf("count=%v win=%q skip=%v err=%v", count, win, skip, err)
	}
	if detail == "" || !strings.Contains(detail, "2026-09-23") {
		t.Fatalf("detail=%q", detail)
	}
	count, detail, _, skip, err = opsEventMetrics("upstream_stale", SyncSnapshot{UpstreamStale: true}, 0, "this_sync")
	if err != nil || skip || count != 1 || !strings.Contains(detail, "stuck") {
		t.Fatalf("no since: detail=%q", detail)
	}
	_, _, _, _, err = opsEventMetrics("nope", SyncSnapshot{}, 0, "this_sync")
	if err == nil {
		t.Fatal("unknown event")
	}
}

func TestOpsEventMetrics_LegacyEvents(t *testing.T) {
	count, _, _, skip, err := opsEventMetrics("repo_fetch_failed", SyncSnapshot{
		ReposFailed: 2, ReposAttempted: 5, FailedRepos: []string{"a/b"},
	}, 0, "this_sync")
	if err != nil || skip || count != 2 {
		t.Fatalf("repo_fetch_failed count=%v skip=%v err=%v", count, skip, err)
	}
	_, _, _, _, err = opsEventMetrics("repo_fetch_failed", SyncSnapshot{}, 0, "bad")
	if err == nil {
		t.Fatal("want window error")
	}

	_, _, _, skip, err = opsEventMetrics("sync_failed", SyncSnapshot{Success: true}, 0, "this_sync")
	if err != nil || !skip {
		t.Fatal("success sync skip")
	}
	count, _, win, skip, err := opsEventMetrics("sync_failed", SyncSnapshot{Success: false}, 0, "this_sync")
	if err != nil || skip || count != 1 || win != "this_sync" {
		t.Fatalf("sync_failed this_sync")
	}
	count, _, _, skip, err = opsEventMetrics("sync_failed", SyncSnapshot{}, 3, "consecutive_runs")
	if err != nil || skip || count != 3 {
		t.Fatalf("consecutive count=%v", count)
	}

	_, _, _, skip, err = opsEventMetrics("github_unreachable", SyncSnapshot{}, 0, "this_sync")
	if err != nil || !skip {
		t.Fatal("reachable skip")
	}
	count, _, _, skip, err = opsEventMetrics("github_unreachable", SyncSnapshot{Unreachable: true}, 0, "this_sync")
	if err != nil || skip || count != 1 {
		t.Fatal("unreachable")
	}

	_, _, _, skip, err = opsEventMetrics("rate_limit", SyncSnapshot{RateLimitRemaining: -1}, 0, "this_sync")
	if err != nil || !skip {
		t.Fatal("no rate limit skip")
	}
	count, _, _, skip, err = opsEventMetrics("rate_limit", SyncSnapshot{RateLimitRemaining: 50}, 0, "this_sync")
	if err != nil || skip || count != 50 {
		t.Fatalf("rate_limit count=%v", count)
	}
}
