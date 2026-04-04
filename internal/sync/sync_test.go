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

func TestResolveReposExplicit(t *testing.T) {
	t.Parallel()
	c := github.NewClient("tok")
	c.BaseURL = "http://should-not-be-called.example"

	repos, err := resolveRepos(c, Options{Repos: []string{"acme/a", "acme/b"}})
	if err != nil {
		t.Fatalf("resolveRepos: %v", err)
	}
	if len(repos) != 2 || repos[0].FullName != "acme/a" || repos[1].FullName != "acme/b" {
		t.Fatalf("got %+v", repos)
	}
}

func TestResolveReposAllUnfiltered(t *testing.T) {
	want := []github.Repo{{FullName: "x/y", StargazersCount: 3}}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/repos" {
			t.Errorf("path = %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := github.NewClient("tok")
	c.BaseURL = srv.URL

	repos, err := resolveRepos(c, Options{Filter: "*"})
	if err != nil {
		t.Fatalf("resolveRepos: %v", err)
	}
	if len(repos) != 1 || repos[0].FullName != "x/y" {
		t.Fatalf("got %+v", repos)
	}
}

func TestResolveReposListReposError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`broken`))
	}))
	defer srv.Close()

	c := github.NewClient("tok")
	c.BaseURL = srv.URL

	_, err := resolveRepos(c, Options{Filter: "*"})
	if err == nil {
		t.Fatal("expected error from ListRepos")
	}
	if !strings.Contains(err.Error(), "list repos") {
		t.Errorf("error = %v", err)
	}
}

func TestResolveReposAppliesFilter(t *testing.T) {
	all := []github.Repo{
		{FullName: "a/1"},
		{FullName: "a/2", Fork: true},
		{FullName: "b/1"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(all)
	}))
	defer srv.Close()

	c := github.NewClient("tok")
	c.BaseURL = srv.URL

	repos, err := resolveRepos(c, Options{Filter: "*,!fork"})
	if err != nil {
		t.Fatalf("resolveRepos: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("got %d repos, want 2", len(repos))
	}
}

func TestRunNoRepos(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]github.Repo{})
	}))
	defer srv.Close()

	c := github.NewClient("tok")
	c.BaseURL = srv.URL
	s := tempStore(t)

	if err := Run(c, s, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRunOneExplicitRepo(t *testing.T) {
	repoPath := "owner/repo"
	ts := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch p {
		case "/repos/" + repoPath + "/pulls":
			json.NewEncoder(w).Encode([]github.PullRequest{})
		case "/repos/" + repoPath + "/traffic/views":
			json.NewEncoder(w).Encode(github.TrafficViews{
				Views: []github.DailyStat{{Timestamp: ts, Count: 5, Uniques: 3}},
			})
		case "/repos/" + repoPath + "/traffic/clones":
			json.NewEncoder(w).Encode(github.TrafficClones{
				Clones: []github.DailyStat{{Timestamp: ts, Count: 2, Uniques: 1}},
			})
		case "/repos/" + repoPath + "/traffic/popular/referrers":
			json.NewEncoder(w).Encode([]github.Referrer{})
		case "/repos/" + repoPath + "/traffic/popular/paths":
			json.NewEncoder(w).Encode([]github.PopularPath{})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, p)
		}
	}))
	defer srv.Close()

	c := github.NewClient("tok")
	c.BaseURL = srv.URL
	s := tempStore(t)

	err := Run(c, s, Options{Repos: []string{repoPath}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	rows, err := s.ViewsByRange(repoPath, "2026-03-20", "2026-03-20")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Count != 5 {
		t.Fatalf("views: %+v", rows)
	}
}

func TestRunWithStarHistory(t *testing.T) {
	repoPath := "owner/repo"
	ts := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case p == "/repos/"+repoPath+"/pulls":
			json.NewEncoder(w).Encode([]github.PullRequest{})
		case p == "/repos/"+repoPath+"/traffic/views":
			json.NewEncoder(w).Encode(github.TrafficViews{
				Views: []github.DailyStat{{Timestamp: ts, Count: 1, Uniques: 1}},
			})
		case p == "/repos/"+repoPath+"/traffic/clones":
			json.NewEncoder(w).Encode(github.TrafficClones{
				Clones: []github.DailyStat{{Timestamp: ts, Count: 1, Uniques: 1}},
			})
		case p == "/repos/"+repoPath+"/traffic/popular/referrers":
			json.NewEncoder(w).Encode([]github.Referrer{})
		case p == "/repos/"+repoPath+"/traffic/popular/paths":
			json.NewEncoder(w).Encode([]github.PopularPath{})
		case strings.HasPrefix(p, "/repos/"+repoPath+"/stargazers"):
			if r.Header.Get("Accept") != "application/vnd.github.v3.star+json" {
				t.Errorf("Accept header = %q", r.Header.Get("Accept"))
			}
			json.NewEncoder(w).Encode([]github.Star{
				{StarredAt: t1},
				{StarredAt: t2},
			})
		default:
			t.Fatalf("unexpected request: %s", p)
		}
	}))
	defer srv.Close()

	c := github.NewClient("tok")
	c.BaseURL = srv.URL
	s := tempStore(t)

	err := Run(c, s, Options{Repos: []string{repoPath}, SyncStars: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	rows, err := s.StarsByRepo(repoPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("stars rows: got %d, want 2", len(rows))
	}
	if rows[0].Total != 1 || rows[1].Total != 2 {
		t.Fatalf("cumulative stars: %+v", rows)
	}
}

func tempStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "sync.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
