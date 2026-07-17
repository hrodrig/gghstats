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

func TestParseRulesJSON_Traffic(t *testing.T) {
	rules, err := ParseRulesJSON(`[
	  {"repo":"a/b","metric":"clones","window":"1d","op":"gte","value":10,"debounce":"once_per_utc_day"}
	]`)
	if err != nil || len(rules) != 1 {
		t.Fatalf("got %+v err=%v", rules, err)
	}
	if rules[0].Kind != KindTraffic {
		t.Fatalf("kind=%q", rules[0].Kind)
	}
}

func TestRunTrafficRules_AbsoluteHighAndDebounce(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	today := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	day := today.Format("2006-01-02")
	if err := db.UpsertRepo("hrodrig/pgwd", "pgwd", 0, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertClone("hrodrig/pgwd", day, 241, 10); err != nil {
		t.Fatal(err)
	}

	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ApplyRetryConfig(RetryConfig{MaxAttempts: 1, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond})
	t.Cleanup(func() { ApplyRetryConfig(DefaultRetryConfig) })

	senders := BuildSenders([]ResolvedSink{{Type: TypeSlack, URL: srv.URL}}, srv.Client())
	rules := []RuleSpec{{
		Kind: KindTraffic, Repo: "hrodrig/pgwd", Metric: "clones", Window: "1d",
		Op: "gte", Value: 225, Debounce: "once_per_utc_day",
	}}

	cfg := EvalConfig{DB: db, Rules: rules, Senders: senders, Now: today, PublicURL: "https://stats.example.com"}
	RunTrafficRules(context.Background(), cfg)
	if len(bodies) != 1 {
		t.Fatalf("want 1 delivery, got %d", len(bodies))
	}
	if !strings.Contains(bodies[0], "hrodrig/pgwd") || !strings.Contains(bodies[0], "241") {
		t.Fatalf("body=%s", bodies[0])
	}

	// Second eval same UTC day — debounced
	RunTrafficRules(context.Background(), cfg)
	if len(bodies) != 1 {
		t.Fatalf("debounce failed, got %d", len(bodies))
	}
}

func TestRunTrafficRules_ZeroClones(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.UpsertRepo("hrodrig/groot", "groot", 0, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	// no clone row → 0

	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ApplyRetryConfig(RetryConfig{MaxAttempts: 1, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond})
	t.Cleanup(func() { ApplyRetryConfig(DefaultRetryConfig) })

	today := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	RunTrafficRules(context.Background(), EvalConfig{
		DB: db,
		Rules: []RuleSpec{{
			Kind: KindTraffic, Repo: "hrodrig/groot", Metric: "clones", Window: "1d",
			Op: "eq", Value: 0, Debounce: "every_sync",
		}},
		Senders: BuildSenders([]ResolvedSink{{Type: TypeSlack, URL: srv.URL}}, srv.Client()),
		Now:     today,
	})
	if n != 1 {
		t.Fatalf("want 1, got %d", n)
	}
}

func TestRunTrafficRules_FleetLifetime(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_ = db.UpsertRepo("a/b", "b", 0, 0, 0, 0, 0, false, false, "")
	_ = db.UpsertClone("a/b", "2026-01-01", 20000, 1)
	_ = db.UpsertClone("a/b", "2026-01-02", 15000, 1)

	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ApplyRetryConfig(RetryConfig{MaxAttempts: 1, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond})
	t.Cleanup(func() { ApplyRetryConfig(DefaultRetryConfig) })

	RunTrafficRules(context.Background(), EvalConfig{
		DB: db,
		Rules: []RuleSpec{{
			Kind: KindTraffic, Scope: "all_repos", Metric: "clones", Window: "lifetime",
			Op: "gte", Value: 30000, Fire: "once",
		}},
		Senders: BuildSenders([]ResolvedSink{{Type: TypeSlack, URL: srv.URL}}, srv.Client()),
		Now:     time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC),
	})
	if n != 1 {
		t.Fatalf("want 1, got %d", n)
	}
	// fire once — second run silent
	RunTrafficRules(context.Background(), EvalConfig{
		DB: db,
		Rules: []RuleSpec{{
			Kind: KindTraffic, Scope: "all_repos", Metric: "clones", Window: "lifetime",
			Op: "gte", Value: 30000, Fire: "once",
		}},
		Senders: BuildSenders([]ResolvedSink{{Type: TypeSlack, URL: srv.URL}}, srv.Client()),
		Now:     time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC),
	})
	if n != 1 {
		t.Fatalf("fire once failed, got %d", n)
	}
}
