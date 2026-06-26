package sync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/github"
	"github.com/hrodrig/gghstats/internal/store"
)

func TestRunRetriesTransientErrorAndSucceeds(t *testing.T) {
	const repoPath = "owner/transient"
	var viewsCalls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/" + repoPath:
			_ = json.NewEncoder(w).Encode(github.Repo{ID: 1, FullName: repoPath, StargazersCount: 1})
		case "/repos/" + repoPath + "/pulls":
			_ = json.NewEncoder(w).Encode([]github.PullRequest{})
		case "/repos/" + repoPath + "/traffic/views":
			if viewsCalls.Add(1) == 1 {
				http.Error(w, "boom", http.StatusBadGateway)
				return
			}
			_ = json.NewEncoder(w).Encode(github.TrafficViews{})
		case "/repos/" + repoPath + "/traffic/clones":
			_ = json.NewEncoder(w).Encode(github.TrafficClones{})
		case "/repos/" + repoPath + "/traffic/popular/referrers":
			_ = json.NewEncoder(w).Encode([]github.Referrer{})
		case "/repos/" + repoPath + "/traffic/popular/paths":
			_ = json.NewEncoder(w).Encode([]github.PopularPath{})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	db, err := store.Open(filepath.Join(t.TempDir(), "retry.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	gh := github.NewClient("tok")
	gh.BaseURL = srv.URL
	gh.SetRetry(github.RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond})

	rec := &fakeRec{kinds: map[string]int{}}
	if err := Run(gh, db, Options{Repos: []string{repoPath}}, rec); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := viewsCalls.Load(); got < 2 {
		t.Fatalf("expected retry on /views, got %d calls", got)
	}
	if rec.kinds["views"] != 0 {
		t.Fatalf("views kind count = %d, want 0 (retry recovered)", rec.kinds["views"])
	}
}
