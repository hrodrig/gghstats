package sync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/github"
	"github.com/hrodrig/gghstats/internal/store"
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
	time.Sleep(80 * time.Millisecond)
	if c.Status().Running {
		t.Fatal("expected repo sync done")
	}
}
