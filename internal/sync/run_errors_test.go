package sync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hrodrig/gghstats/internal/github"
	"github.com/hrodrig/gghstats/internal/store"
)

type fakeRec struct {
	mu    sync.Mutex
	kinds map[string]int
	repos map[string]int
}

func (f *fakeRec) ObserveSyncError(kind string) {
	f.mu.Lock()
	f.kinds[kind]++
	f.mu.Unlock()
}

func (f *fakeRec) ObserveSyncRepo(status string) {
	f.mu.Lock()
	f.repos[status]++
	f.mu.Unlock()
}

func TestRunClassifiesViewsFailure(t *testing.T) {
	const repoPath = "owner/views-fail"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/" + repoPath:
			_ = json.NewEncoder(w).Encode(github.Repo{ID: 1, FullName: repoPath, StargazersCount: 1})
		case "/repos/" + repoPath + "/pulls":
			_ = json.NewEncoder(w).Encode([]github.PullRequest{})
		case "/repos/" + repoPath + "/traffic/views":
			http.Error(w, "boom", http.StatusBadGateway)
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

	db, err := store.Open(filepath.Join(t.TempDir(), "views.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	gh := github.NewClient("tok")
	gh.BaseURL = srv.URL
	rec := &fakeRec{kinds: map[string]int{}, repos: map[string]int{}}

	if _, err := Run(gh, db, Options{Repos: []string{repoPath}}, rec); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rec.kinds["views"] != 1 {
		t.Fatalf("views kind count = %d, want 1 (got %v)", rec.kinds["views"], rec.kinds)
	}
}

func TestRunRepoMetaFailureClassified(t *testing.T) {
	const repoPath = "owner/meta-fail"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user/repos":
			_ = json.NewEncoder(w).Encode([]github.Repo{{FullName: repoPath}})
		case "/repos/" + repoPath:
			http.Error(w, "nope", http.StatusNotFound)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	db, err := store.Open(filepath.Join(t.TempDir(), "meta.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	gh := github.NewClient("tok")
	gh.BaseURL = srv.URL
	rec := &fakeRec{kinds: map[string]int{}, repos: map[string]int{}}

	_, _ = Run(gh, db, Options{}, rec)
	if rec.kinds["worker"] < 1 {
		t.Fatalf("worker kind count = %d, want >= 1 (got %v)", rec.kinds["worker"], rec.kinds)
	}
	if rec.kinds["repo_meta"] != 1 {
		t.Fatalf("repo_meta kind count = %d, want 1 (got %v)", rec.kinds["repo_meta"], rec.kinds)
	}
}
