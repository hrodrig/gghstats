package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/github"
	"github.com/hrodrig/gghstats/internal/sync"
)

func testSyncHandler(t *testing.T) http.Handler {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/repos" {
			_ = json.NewEncoder(w).Encode([]github.Repo{})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	db := testStore(t)
	gh := github.NewClient("tok")
	gh.BaseURL = srv.URL
	coord := sync.NewCoordinator(gh, db, sync.Options{Filter: "*"})
	return New(Config{Store: db, SyncCoordinator: coord, APIToken: "secret", DisableMetrics: true})
}

func TestAPISyncDisabledWithoutToken(t *testing.T) {
	db := testStore(t)
	coord := sync.NewCoordinator(github.NewClient("tok"), db, sync.Options{Filter: "*"})
	handler := New(Config{Store: db, SyncCoordinator: coord, APIToken: "", DisableMetrics: true})

	req := httptest.NewRequest("GET", "/api/v1/sync", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestAPISyncUnauthorized(t *testing.T) {
	handler := testSyncHandler(t)

	req := httptest.NewRequest("GET", "/api/v1/sync", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestAPISyncStatusAndStart(t *testing.T) {
	handler := testSyncHandler(t)

	req := httptest.NewRequest("GET", "/api/v1/sync", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("GET status = %d", w.Code)
	}

	req2 := httptest.NewRequest("POST", "/api/v1/sync", nil)
	req2.Header.Set("x-api-token", "secret")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != 202 {
		t.Fatalf("POST status = %d, want 202", w2.Code)
	}

	time.Sleep(80 * time.Millisecond)

	req3 := httptest.NewRequest("GET", "/api/v1/sync", nil)
	req3.Header.Set("x-api-token", "secret")
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)

	var st struct {
		Running      bool   `json:"running"`
		LastFinished string `json:"last_finished_at"`
		LastError    string `json:"last_error"`
	}
	if err := json.NewDecoder(w3.Body).Decode(&st); err != nil {
		t.Fatal(err)
	}
	if st.Running {
		t.Fatal("expected sync finished")
	}
	if st.LastFinished == "" {
		t.Fatalf("status = %+v", st)
	}
}

func TestAPISyncConflictWhenRunning(t *testing.T) {
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

	db := testStore(t)
	gh := github.NewClient("tok")
	gh.BaseURL = srv.URL
	coord := sync.NewCoordinator(gh, db, sync.Options{Filter: "*"})
	handler := New(Config{Store: db, SyncCoordinator: coord, APIToken: "secret", DisableMetrics: true})

	if err := coord.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)

	req := httptest.NewRequest("POST", "/api/v1/sync", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 409 {
		t.Fatalf("status = %d, want 409", w.Code)
	}
	if !strings.Contains(w.Body.String(), "sync_in_progress") {
		t.Fatalf("body = %q", w.Body.String())
	}
	close(block)
}

func TestIndexShowsSyncButtonWhenAPIEnabled(t *testing.T) {
	handler := testSyncHandler(t)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), `id="sync-now-btn"`) {
		t.Error("expected Sync now button when API token configured")
	}
	if !strings.Contains(w.Body.String(), "Sync all") {
		t.Error("expected Sync all label on index")
	}
}

func TestAPISyncStartSingleRepo(t *testing.T) {
	const repo = "o/r"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/"+repo:
			_ = json.NewEncoder(w).Encode(github.Repo{ID: 1, FullName: repo})
		case strings.HasSuffix(r.URL.Path, "/traffic/views"):
			_ = json.NewEncoder(w).Encode(github.TrafficViews{})
		case strings.HasSuffix(r.URL.Path, "/traffic/clones"):
			_ = json.NewEncoder(w).Encode(github.TrafficClones{})
		case strings.HasSuffix(r.URL.Path, "/traffic/popular/referrers"):
			_ = json.NewEncoder(w).Encode([]github.Referrer{})
		case strings.HasSuffix(r.URL.Path, "/traffic/popular/paths"):
			_ = json.NewEncoder(w).Encode([]github.PopularPath{})
		case strings.HasSuffix(r.URL.Path, "/pulls"):
			_ = json.NewEncoder(w).Encode([]github.PullRequest{})
		case strings.HasSuffix(r.URL.Path, "/stargazers"):
			_ = json.NewEncoder(w).Encode([]github.Star{})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	db := testStore(t)
	gh := github.NewClient("tok")
	gh.BaseURL = srv.URL
	coord := sync.NewCoordinator(gh, db, sync.Options{Filter: "*"})
	handler := New(Config{Store: db, SyncCoordinator: coord, APIToken: "secret", DisableMetrics: true})

	req := httptest.NewRequest("POST", "/api/v1/sync?repo="+repo, nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 202 {
		t.Fatalf("POST status = %d body=%s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["scope"] != "repo" || body["repo"] != repo {
		t.Fatalf("body = %+v", body)
	}

	time.Sleep(100 * time.Millisecond)
	st := coord.Status()
	if st.Running {
		t.Fatal("expected single-repo sync finished")
	}
	if st.Scope != "" && st.Repo != "" {
		// cleared after finish; last run metadata not persisted in status
	}
}
