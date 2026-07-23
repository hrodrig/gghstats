package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestDogfoodContract_APIOnly rebuilds index, repo, and H2H from documented JSON endpoints alone.
func TestDogfoodContract_APIOnly(t *testing.T) {
	db := seedH2HRepos(t)
	today := time.Now().UTC().Format("2006-01-02")
	_ = db.UpsertStar("a/one", today, 10)
	_ = db.UpsertStar("b/two", today, 5)

	const token = "dogfood-token"
	h := New(Config{
		Store:          db,
		APIToken:       token,
		APIOnly:        true,
		DisableMetrics: true,
	})

	getJSON := func(path string) map[string]interface{} {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("x-api-token", token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, w.Code, w.Body.String())
		}
		var out map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("%s decode: %v", path, err)
		}
		return out
	}

	// Index
	index := getJSON("/api/repos?sort=name&dir=asc")
	if index["total_count"].(float64) < 2 {
		t.Fatalf("index total_count = %v", index["total_count"])
	}
	if _, ok := index["items"]; !ok {
		t.Fatal("index missing items")
	}
	chart := getJSON("/api/v1/charts/index-clones")
	if _, ok := chart["series"]; !ok {
		t.Fatal("chart missing series")
	}

	// Repo page dogfood
	repo := getJSON("/api/v1/repos/a/one")
	if repo["repo"] == nil {
		t.Fatal("repo missing summary")
	}
	if _, ok := repo["momentum_7d"]; !ok {
		t.Fatal("repo missing momentum_7d")
	}
	traffic := getJSON("/api/v1/repos/a/one/traffic?days=30")
	if traffic["clones"] == nil || traffic["views"] == nil {
		t.Fatal("traffic missing series")
	}
	stars := getJSON("/api/v1/repos/a/one/stars")
	if stars["stars"] == nil {
		t.Fatal("stars missing")
	}
	popular := getJSON("/api/v1/repos/a/one/popular")
	if popular["referrers"] == nil || popular["paths"] == nil {
		t.Fatal("popular missing")
	}

	// H2H
	h2hResp := getJSON("/api/v1/h2h?a=a/one&b=b/two&w=7d")
	if h2hResp["result"] == nil {
		t.Fatal("h2h missing result")
	}
	if h2hResp["charts"] == nil {
		t.Fatal("h2h missing charts")
	}
}
