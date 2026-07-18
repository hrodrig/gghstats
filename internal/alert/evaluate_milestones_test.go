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

func TestParseRulesJSON_Milestones(t *testing.T) {
	rules, err := ParseRulesJSON(`[
	  {"repo":"hrodrig/pgwd","metric":"stars","milestones":[500,100,100],"fire":"once"}
	]`)
	if err != nil || len(rules) != 1 {
		t.Fatalf("got %+v err=%v", rules, err)
	}
	r := rules[0]
	if r.Metric != "stars" || r.Window != "milestone" || r.Debounce != "once" {
		t.Fatalf("normalized=%+v", r)
	}
	if len(r.Milestones) != 2 || r.Milestones[0] != 100 || r.Milestones[1] != 500 {
		t.Fatalf("milestones=%v want sorted deduped [100 500]", r.Milestones)
	}
}

func TestParseRulesJSON_MilestonesReject(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"bad_metric", `[{"repo":"a/b","metric":"clones","milestones":[100]}]`},
		{"no_repo", `[{"metric":"stars","milestones":[100]}]`},
		{"non_positive", `[{"repo":"a/b","metric":"stars","milestones":[0]}]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseRulesJSON(tc.raw); err == nil {
				t.Fatal("want error")
			}
		})
	}
}

func TestRunMilestoneRules_LadderAndDebounce(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.UpsertRepo("hrodrig/pgwd", "pgwd", 120, 0, 0, 0, 0, false, false, ""); err != nil {
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

	rules, err := ParseRulesJSON(`[{"repo":"hrodrig/pgwd","metric":"stars","milestones":[100,500]}]`)
	if err != nil {
		t.Fatal(err)
	}
	senders := BuildSenders([]ResolvedSink{{Type: TypeSlack, URL: srv.URL}}, srv.Client())
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	cfg := EvalConfig{DB: db, Rules: rules, Senders: senders, Now: now, PublicURL: "https://stats.example.com"}

	RunMilestoneRules(context.Background(), cfg)
	if len(bodies) != 1 {
		t.Fatalf("want 1 (crossed 100 only), got %d bodies=%v", len(bodies), bodies)
	}
	if !strings.Contains(bodies[0], "crossed 100 (next 500)") {
		t.Fatalf("body=%s", bodies[0])
	}
	if !strings.Contains(bodies[0], "window:   milestone") {
		t.Fatalf("missing window: %s", bodies[0])
	}

	RunMilestoneRules(context.Background(), cfg)
	if len(bodies) != 1 {
		t.Fatalf("100 should debounce once, got %d", len(bodies))
	}

	if err := db.UpsertRepo("hrodrig/pgwd", "pgwd", 600, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	RunMilestoneRules(context.Background(), cfg)
	if len(bodies) != 2 {
		t.Fatalf("want 2 after crossing 500, got %d", len(bodies))
	}
	if !strings.Contains(bodies[1], "crossed 500 (final)") {
		t.Fatalf("second body=%s", bodies[1])
	}

	RunMilestoneRules(context.Background(), cfg)
	if len(bodies) != 2 {
		t.Fatalf("500 should debounce once, got %d", len(bodies))
	}
}

func TestRunAllRules_MilestonesOnSuccessOnly(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.UpsertRepo("hrodrig/pgwd", "pgwd", 200, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}

	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ApplyRetryConfig(RetryConfig{MaxAttempts: 1, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond})
	t.Cleanup(func() { ApplyRetryConfig(DefaultRetryConfig) })

	rules, err := ParseRulesJSON(`[{"repo":"hrodrig/pgwd","metric":"stars","milestones":[100]}]`)
	if err != nil {
		t.Fatal(err)
	}
	cfg := EvalConfig{
		DB: db, Rules: rules,
		Senders: BuildSenders([]ResolvedSink{{Type: TypeSlack, URL: srv.URL}}, srv.Client()),
		Now:     time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC),
	}

	RunAllRules(context.Background(), cfg, SyncSnapshot{Success: false})
	if n != 0 {
		t.Fatalf("failed sync should not run milestones, got %d", n)
	}
	RunAllRules(context.Background(), cfg, SyncSnapshot{Success: true})
	if n != 1 {
		t.Fatalf("success sync want 1, got %d", n)
	}
}
