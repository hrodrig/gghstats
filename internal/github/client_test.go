package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestViews(t *testing.T) {
	want := TrafficViews{
		Count:   42,
		Uniques: 10,
		Views: []DailyStat{
			{Timestamp: time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC), Count: 20, Uniques: 5},
			{Timestamp: time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC), Count: 22, Uniques: 8},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/traffic/views" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("authorization = %q, want Bearer test-token", got)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := NewClient("test-token")
	c.BaseURL = srv.URL

	got, err := c.Views("owner/repo")
	if err != nil {
		t.Fatalf("Views() error: %v", err)
	}
	if got.Count != want.Count {
		t.Errorf("Count = %d, want %d", got.Count, want.Count)
	}
	if len(got.Views) != 2 {
		t.Fatalf("len(Views) = %d, want 2", len(got.Views))
	}
}

func TestClones(t *testing.T) {
	want := TrafficClones{
		Count:   100,
		Uniques: 30,
		Clones: []DailyStat{
			{Timestamp: time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC), Count: 60, Uniques: 20},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := NewClient("tok")
	c.BaseURL = srv.URL

	got, err := c.Clones("owner/repo")
	if err != nil {
		t.Fatalf("Clones() error: %v", err)
	}
	if got.Count != 100 {
		t.Errorf("Count = %d, want 100", got.Count)
	}
}

func TestReferrers(t *testing.T) {
	want := []Referrer{
		{Referrer: "google.com", Count: 50, Uniques: 10},
		{Referrer: "github.com", Count: 30, Uniques: 8},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := NewClient("tok")
	c.BaseURL = srv.URL

	got, err := c.Referrers("owner/repo")
	if err != nil {
		t.Fatalf("Referrers() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Referrer != "google.com" {
		t.Errorf("got %q, want google.com", got[0].Referrer)
	}
}

func TestPopularPaths(t *testing.T) {
	want := []PopularPath{
		{Path: "/hrodrig/pgwd", Title: "pgwd", Count: 100, Uniques: 20},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := NewClient("tok")
	c.BaseURL = srv.URL

	got, err := c.PopularPaths("owner/repo")
	if err != nil {
		t.Fatalf("PopularPaths() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
}

func TestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"Bad credentials"}`))
	}))
	defer srv.Close()

	c := NewClient("bad-token")
	c.BaseURL = srv.URL

	_, err := c.Views("owner/repo")
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestListRepos(t *testing.T) {
	repos := []Repo{
		{ID: 1, FullName: "owner/a", StargazersCount: 10},
		{ID: 2, FullName: "owner/b", StargazersCount: 5},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(repos)
	}))
	defer srv.Close()

	c := NewClient("tok")
	c.BaseURL = srv.URL

	got, err := c.ListRepos(false)
	if err != nil {
		t.Fatalf("ListRepos() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].FullName != "owner/a" {
		t.Errorf("got %q, want owner/a", got[0].FullName)
	}
}

func TestListReposPaginated(t *testing.T) {
	page1 := []Repo{{ID: 1, FullName: "a/1"}}
	page2 := []Repo{{ID: 2, FullName: "a/2"}}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			json.NewEncoder(w).Encode(page2)
			return
		}
		nextURL := "http://" + r.Host + "/user/repos?visibility=public&per_page=100&page=2"
		w.Header().Set("Link", `<`+nextURL+`>; rel="next"`)
		json.NewEncoder(w).Encode(page1)
	}))
	defer srv.Close()

	c := NewClient("tok")
	c.BaseURL = srv.URL

	got, err := c.ListRepos(false)
	if err != nil {
		t.Fatalf("ListRepos() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (paginated)", len(got))
	}
}

func TestStargazers(t *testing.T) {
	stars := []Star{
		{StarredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{StarredAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github.v3.star+json" {
			t.Errorf("wrong accept header: %s", r.Header.Get("Accept"))
		}
		json.NewEncoder(w).Encode(stars)
	}))
	defer srv.Close()

	c := NewClient("tok")
	c.BaseURL = srv.URL

	got, err := c.Stargazers("owner/repo")
	if err != nil {
		t.Fatalf("Stargazers() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}

func TestRepo(t *testing.T) {
	want := Repo{
		FullName:        "you/book",
		StargazersCount: 3,
		Fork:            true,
		Parent:          &RepoParent{FullName: "rust-lang/book"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/you/book" {
			t.Errorf("path = %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := NewClient("tok")
	c.BaseURL = srv.URL

	got, err := c.Repo("you/book")
	if err != nil {
		t.Fatalf("Repo: %v", err)
	}
	if got.FullName != want.FullName || got.ParentFullName() != "rust-lang/book" {
		t.Fatalf("got %+v", got)
	}
}

func TestOpenPullRequests(t *testing.T) {
	want := []PullRequest{{ID: 101}, {ID: 102}}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/pulls" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := NewClient("tok")
	c.BaseURL = srv.URL

	got, err := c.OpenPullRequests("owner/repo")
	if err != nil {
		t.Fatalf("OpenPullRequests: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != 101 {
		t.Errorf("ID = %d", got[0].ID)
	}
}

func TestNextPagePath(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"", ""},
		{`<https://api.github.com/user/repos?page=2>; rel="next"`, "/user/repos?page=2"},
		{`<https://api.github.com/user/repos?page=2>; rel="next", <https://api.github.com/user/repos?page=5>; rel="last"`, "/user/repos?page=2"},
		{`<https://api.github.com/user/repos?page=5>; rel="last"`, ""},
	}
	for _, tt := range tests {
		got := nextPagePath(tt.header)
		if got != tt.want {
			t.Errorf("nextPagePath(%q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}
