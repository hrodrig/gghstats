package sync

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/github"
	"github.com/hrodrig/gghstats/internal/metrics"
	"github.com/hrodrig/gghstats/internal/store"
	"github.com/prometheus/client_golang/prometheus"
)

func testCoordinator(t *testing.T) *Coordinator {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/repos" {
			_ = json.NewEncoder(w).Encode([]github.Repo{})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	db, err := store.Open(filepath.Join(t.TempDir(), "coord.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	gh := github.NewClient("tok")
	gh.BaseURL = srv.URL
	return NewCoordinator(gh, db, Options{Filter: "*"})
}

func TestCoordinatorRunAndStatus(t *testing.T) {
	c := testCoordinator(t)
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	st := c.Status()
	if st.Running || st.LastStartedAt == nil || st.LastFinishedAt == nil {
		t.Fatalf("status = %+v", st)
	}
}

func TestCoordinatorStartRejectsOverlap(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/repos" {
			<-block
			_ = json.NewEncoder(w).Encode([]github.Repo{})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	db, _ := store.Open(filepath.Join(t.TempDir(), "slow.db"))
	defer db.Close()
	gh := github.NewClient("tok")
	gh.BaseURL = srv.URL
	slow := NewCoordinator(gh, db, Options{Filter: "*"})

	if err := slow.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if err := slow.Start(); err != ErrInProgress {
		t.Fatalf("Start() = %v, want ErrInProgress", err)
	}
	if err := slow.Run(); err != ErrInProgress {
		t.Fatalf("Run() = %v, want ErrInProgress", err)
	}
	close(block)
	time.Sleep(50 * time.Millisecond)
}

func TestCoordinatorConcurrentRunSingleFlight(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/repos" {
			<-block
			_ = json.NewEncoder(w).Encode([]github.Repo{})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	db, _ := store.Open(filepath.Join(t.TempDir(), "race.db"))
	defer db.Close()
	gh := github.NewClient("tok")
	gh.BaseURL = srv.URL
	c := NewCoordinator(gh, db, Options{Filter: "*"})

	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := c.Run(); err != ErrInProgress {
		t.Fatalf("Run() = %v, want ErrInProgress", err)
	}
	close(block)
	time.Sleep(50 * time.Millisecond)
}

func TestCoordinatorStartRepo(t *testing.T) {
	const repo = "a/one"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/"+repo {
			_ = json.NewEncoder(w).Encode(github.Repo{ID: 42, FullName: repo})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/pulls") {
			_ = json.NewEncoder(w).Encode([]github.PullRequest{})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/repos/"+repo+"/traffic/") {
			if strings.HasSuffix(r.URL.Path, "/views") {
				_ = json.NewEncoder(w).Encode(github.TrafficViews{})
			} else if strings.HasSuffix(r.URL.Path, "/clones") {
				_ = json.NewEncoder(w).Encode(github.TrafficClones{})
			} else if strings.HasSuffix(r.URL.Path, "/referrers") {
				_ = json.NewEncoder(w).Encode([]github.Referrer{})
			} else if strings.HasSuffix(r.URL.Path, "/paths") {
				_ = json.NewEncoder(w).Encode([]github.PopularPath{})
			}
			return
		}
		if strings.HasSuffix(r.URL.Path, "/stargazers") {
			_ = json.NewEncoder(w).Encode([]github.Star{})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	db, _ := store.Open(filepath.Join(t.TempDir(), "one.db"))
	defer db.Close()
	gh := github.NewClient("tok")
	gh.BaseURL = srv.URL
	c := NewCoordinator(gh, db, Options{Filter: "*"})

	if err := c.StartRepo(repo); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		st := c.Status()
		if !st.Running {
			if st.LastFinishedAt == nil {
				t.Fatalf("status = %+v, want finished sync", st)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected repo sync done within timeout")
}

func TestCoordinatorSetMetricsObserveSync(t *testing.T) {
	reg := prometheus.NewRegistry()
	dom := metrics.RegisterDomain(reg, metrics.DomainConfig{})
	c := testCoordinator(t)
	c.SetMetrics(dom)

	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	if !coordinatorHasSyncSample(reg, "success") {
		t.Fatal("expected success sync duration after Run")
	}
	st := c.Status()
	if st.LastError != "" {
		t.Fatalf("LastError = %q, want empty", st.LastError)
	}
}

func TestCoordinatorObserveSyncOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/repos" {
			http.Error(w, "upstream down", http.StatusInternalServerError)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	db, err := store.Open(filepath.Join(t.TempDir(), "fail.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	gh := github.NewClient("tok")
	gh.BaseURL = srv.URL
	reg := prometheus.NewRegistry()
	dom := metrics.RegisterDomain(reg, metrics.DomainConfig{})
	c := NewCoordinator(gh, db, Options{Filter: "*"})
	c.SetMetrics(dom)

	if err := c.Run(); err == nil {
		t.Fatal("expected sync error")
	}
	if !coordinatorHasSyncSample(reg, "error") {
		t.Fatal("expected error sync duration after failed Run")
	}
	if c.Status().LastError == "" {
		t.Fatal("expected LastError on failed sync")
	}
}

func TestCoordinatorFinishRunDirect(t *testing.T) {
	reg := prometheus.NewRegistry()
	dom := metrics.RegisterDomain(reg, metrics.DomainConfig{})
	c := NewCoordinator(nil, nil, Options{})
	c.SetMetrics(dom)

	c.finishRun(RunResult{Success: true}, nil)
	if coordinatorHasSyncSample(reg, "success") {
		t.Fatal("finishRun without markRunning should not observe sync")
	}

	c.markRunningLocked("all", "")
	c.finishRun(RunResult{Success: false}, errors.New("sync failed"))
	if !coordinatorHasSyncSample(reg, "error") {
		t.Fatal("expected error sync observation")
	}
	if c.Status().LastError != "sync failed" {
		t.Fatalf("LastError = %q", c.Status().LastError)
	}
}

func TestCoordinatorSetAfterSync(t *testing.T) {
	c := NewCoordinator(nil, nil, Options{})
	called := false
	c.SetAfterSync(func(RunResult) { called = true })
	c.mu.Lock()
	fn := c.afterSync
	c.mu.Unlock()
	if fn == nil {
		t.Fatal("afterSync not set")
	}
	fn(RunResult{Success: true})
	if !called {
		t.Fatal("callback not invoked")
	}
}

func coordinatorHasSyncSample(reg *prometheus.Registry, status string) bool {
	mfs, err := reg.Gather()
	if err != nil {
		return false
	}
	for _, mf := range mfs {
		if mf.GetName() != "gghstats_sync_duration_seconds" {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == "status" && lp.GetValue() == status && m.GetHistogram().GetSampleCount() > 0 {
					return true
				}
			}
		}
	}
	return false
}
