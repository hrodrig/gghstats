package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/github"
	"github.com/hrodrig/gghstats/internal/store"
)

func TestUpsertRepoFromGitHub(t *testing.T) {
	want := github.Repo{
		FullName:        "o/r",
		StargazersCount: 42,
		OpenIssuesCount: 3,
		ForksCount:      1,
		WatchersCount:   2,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/o/r":
			json.NewEncoder(w).Encode(want)
		case "/repos/o/r/pulls":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := github.NewClient("tok")
	c.BaseURL = srv.URL
	s := tempFetchStore(t)

	if err := upsertRepoFromGitHub(c, s, "o/r"); err != nil {
		t.Fatal(err)
	}
	r, err := s.RepoByName("o/r")
	if err != nil || r == nil {
		t.Fatalf("RepoByName: %v", err)
	}
	if r.Stars != 42 || r.Issues != 3 {
		t.Fatalf("stored metadata: %+v", r)
	}
}

func TestUpsertRepoFromGitHubClampsNegativeIssues(t *testing.T) {
	meta := github.Repo{
		FullName:        "o/r",
		OpenIssuesCount: 1,
		StargazersCount: 0,
		ForksCount:      0,
		WatchersCount:   0,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/o/r":
			json.NewEncoder(w).Encode(meta)
		case "/repos/o/r/pulls":
			json.NewEncoder(w).Encode([]github.PullRequest{{ID: 1}, {ID: 2}})
		default:
			t.Fatalf("unexpected %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := github.NewClient("tok")
	c.BaseURL = srv.URL
	s := tempFetchStore(t)

	if err := upsertRepoFromGitHub(c, s, "o/r"); err != nil {
		t.Fatal(err)
	}
	r, err := s.RepoByName("o/r")
	if err != nil || r == nil {
		t.Fatalf("RepoByName: %v", err)
	}
	if r.Issues != 0 {
		t.Errorf("issues = %d, want 0 (clamped)", r.Issues)
	}
}

func TestRunFetchEndToEnd(t *testing.T) {
	repo := "owner/repo"
	ts := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch p {
		case "/repos/" + repo:
			json.NewEncoder(w).Encode(github.Repo{
				FullName: repo, StargazersCount: 1, OpenIssuesCount: 0,
				ForksCount: 0, WatchersCount: 1,
			})
		case "/repos/" + repo + "/pulls":
			json.NewEncoder(w).Encode([]github.PullRequest{})
		case "/repos/" + repo + "/traffic/views":
			json.NewEncoder(w).Encode(github.TrafficViews{
				Views: []github.DailyStat{{Timestamp: ts, Count: 7, Uniques: 4}},
			})
		case "/repos/" + repo + "/traffic/clones":
			json.NewEncoder(w).Encode(github.TrafficClones{
				Clones: []github.DailyStat{{Timestamp: ts, Count: 2, Uniques: 1}},
			})
		case "/repos/" + repo + "/traffic/popular/referrers":
			json.NewEncoder(w).Encode([]github.Referrer{})
		case "/repos/" + repo + "/traffic/popular/paths":
			json.NewEncoder(w).Encode([]github.PopularPath{})
		default:
			t.Fatalf("unexpected %s", p)
		}
	}))
	defer srv.Close()

	t.Setenv("GGHSTATS_GITHUB_API_BASE_URL", srv.URL)

	dbPath := filepath.Join(t.TempDir(), "fetch-e2e.db")
	if err := runFetch([]string{"-repo", repo, "-token", "tok", "-db", dbPath}); err != nil {
		t.Fatalf("runFetch: %v", err)
	}

	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	v, _ := st.ViewsByRange(repo, "2026-04-01", "2026-04-01")
	if len(v) != 1 || v[0].Count != 7 {
		t.Fatalf("views after fetch: %+v", v)
	}
}

func TestFetchStoreViewsClonesReferrersPaths(t *testing.T) {
	ts := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	today := "2026-04-04"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch p {
		case "/repos/x/y/traffic/views":
			json.NewEncoder(w).Encode(github.TrafficViews{
				Count:   2,
				Uniques: 1,
				Views:   []github.DailyStat{{Timestamp: ts, Count: 10, Uniques: 5}},
			})
		case "/repos/x/y/traffic/clones":
			json.NewEncoder(w).Encode(github.TrafficClones{
				Count:   2,
				Uniques: 1,
				Clones:  []github.DailyStat{{Timestamp: ts, Count: 3, Uniques: 2}},
			})
		case "/repos/x/y/traffic/popular/referrers":
			json.NewEncoder(w).Encode([]github.Referrer{{Referrer: "ex.com", Count: 1, Uniques: 1}})
		case "/repos/x/y/traffic/popular/paths":
			json.NewEncoder(w).Encode([]github.PopularPath{{Path: "/p", Title: "P", Count: 2, Uniques: 2}})
		default:
			t.Fatalf("unexpected %s", p)
		}
	}))
	defer srv.Close()

	c := github.NewClient("tok")
	c.BaseURL = srv.URL
	s := tempFetchStore(t)
	repo := "x/y"

	if err := fetchStoreViews(c, s, repo); err != nil {
		t.Fatal(err)
	}
	if err := fetchStoreClones(c, s, repo); err != nil {
		t.Fatal(err)
	}
	if err := fetchStoreReferrers(c, s, repo, today); err != nil {
		t.Fatal(err)
	}
	if err := fetchStorePaths(c, s, repo, today); err != nil {
		t.Fatal(err)
	}

	v, _ := s.ViewsByRange(repo, "2026-04-01", "2026-04-01")
	if len(v) != 1 || v[0].Count != 10 {
		t.Fatalf("views: %+v", v)
	}
	cl, _ := s.ClonesByRange(repo, "2026-04-01", "2026-04-01")
	if len(cl) != 1 || cl[0].Count != 3 {
		t.Fatalf("clones: %+v", cl)
	}
}

func tempFetchStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
