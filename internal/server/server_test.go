package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hrodrig/gghstats/internal/store"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestHealthEndpoint(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db})

	req := httptest.NewRequest("GET", HealthzPath, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if w.Body.String() != `{"status":"ok"}` {
		t.Errorf("body = %q", w.Body.String())
	}
}

func TestMetricsEndpoint(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db})

	// First scrape runs before the outer middleware records the /metrics request; scrape twice
	// so counters and histograms appear in the body.
	for range 2 {
		req := httptest.NewRequest("GET", MetricsPath, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	}
	req := httptest.NewRequest("GET", MetricsPath, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	body := w.Body.String()

	if !strings.Contains(body, "gghstats_build_info") {
		preview := body
		if len(preview) > 200 {
			preview = preview[:200]
		}
		t.Fatalf("expected gghstats_build_info in exposition, got %q", preview)
	}
	if !strings.Contains(body, "gghstats_http_requests_total") {
		t.Fatal("expected gghstats_http_requests_total in exposition")
	}
}

func TestMetricsDisabled(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db, DisableMetrics: true})

	req := httptest.NewRequest("GET", MetricsPath, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestMainStylesheetEmbedded(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db})

	req := httptest.NewRequest("GET", "/static/bootstrap.min.css", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if len(body) < 1000 || !strings.Contains(body, "Bootstrap") {
		t.Fatalf("expected Bootstrap CSS body, got %d bytes", len(body))
	}
}

func TestFaviconEmbeddedFromAssets(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db})

	for _, path := range []string{
		"/static/favicon.ico",
		"/static/favicon.svg",
		"/static/favicon-16x16.png",
		"/static/favicon-32x32.png",
		"/static/apple-touch-icon.png",
		"/static/android-chrome-192x192.png",
		"/static/android-chrome-512x512.png",
		"/static/manifest.json",
	} {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", path, w.Code)
		}
	}
	req := httptest.NewRequest("GET", "/static/favicon.svg", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), "<svg") {
		t.Error("expected SVG body for favicon.svg")
	}

	req = httptest.NewRequest("GET", "/static/manifest.json", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/manifest+json") {
		t.Errorf("manifest Content-Type = %q, want application/manifest+json…", ct)
	}
	if !strings.Contains(w.Body.String(), `"name"`) || !strings.Contains(w.Body.String(), "gghstats") {
		t.Error("expected Web manifest JSON body")
	}
}

func TestIndexPage(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "test repo", 10, 2, 10, 1, 0, false, false, "")
	db.UpsertView("a/b", "2026-03-20", 50, 20)
	_ = db.UpsertClone("a/b", "2026-03-18", 5, 2)
	_ = db.UpsertClone("a/b", "2026-03-19", 12, 4)

	handler := New(Config{Store: db})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if len(body) < 100 {
		t.Error("response too short, expected HTML page")
	}
	if !strings.Contains(body, `/static/bootstrap.min.css`) {
		t.Error("expected embedded Bootstrap stylesheet link in HTML")
	}
	if !strings.Contains(body, "Neobrutalist") {
		t.Error("expected neobrutalist UI label in footer")
	}
	if !strings.Contains(body, `offcanvas-lg`) || !strings.Contains(body, "Repositories") {
		t.Error("expected app shell layout (sidebar + title)")
	}
	if !strings.Contains(body, "total across list") || !strings.Contains(body, ">10<") {
		t.Error("expected KPI summary for seeded repo (10 stars)")
	}
	if !strings.Contains(body, `id="chart_index_clones"`) {
		t.Error("expected index clones-over-time chart canvas")
	}
	if !strings.Contains(body, "gghstatsListClonesData") || !strings.Contains(body, "2026-03-18") {
		t.Error("expected embedded clone series JSON for chart")
	}
}

func TestIndexHidesPaginationWhenOnePage(t *testing.T) {
	db := testStore(t)
	_ = db.UpsertRepo("a/b", "x", 1, 0, 1, 0, 0, false, false, "")
	handler := New(Config{Store: db})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if strings.Contains(w.Body.String(), "app-repo-pagination-bar") {
		t.Fatal("pagination should be hidden when total <= per_page")
	}
}

func TestIndexPagePagination(t *testing.T) {
	db := testStore(t)
	for i := 0; i < 30; i++ {
		name := fmt.Sprintf("owner/repo-%02d", i)
		_ = db.UpsertRepo(name, "test", i, 0, i, 0, 0, false, false, "")
	}
	handler := New(Config{Store: db})

	req := httptest.NewRequest("GET", "/?per_page=10&page=2&sort=name&dir=asc", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `<strong>11</strong>–<strong>20</strong> of <strong>30</strong> repositories`) {
		t.Fatalf("unexpected pagination summary: %s", body)
	}
	if !strings.Contains(body, "Page 2") {
		t.Fatalf("expected page indicator in response")
	}
}

func TestIndexInvalidPageAndPerPageQuery(t *testing.T) {
	db := testStore(t)
	_ = db.UpsertRepo("a/b", "x", 1, 0, 1, 0, 0, false, false, "")
	handler := New(Config{Store: db})

	req := httptest.NewRequest("GET", "/?page=abc&per_page=-99", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200 (fallback pagination)", w.Code)
	}
}

func TestIndexPerPageCappedAtMax(t *testing.T) {
	db := testStore(t)
	for i := 0; i < 150; i++ {
		_ = db.UpsertRepo(fmt.Sprintf("o/r-%03d", i), "x", 1, 0, 1, 0, 0, false, false, "")
	}
	handler := New(Config{Store: db})

	req := httptest.NewRequest("GET", "/?per_page=500", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Showing <strong>1</strong>–<strong>100</strong> of <strong>150</strong>") {
		t.Fatal("expected per_page capped at 100 in summary")
	}
}

func TestExecuteTemplateMissingName(t *testing.T) {
	tmpl := template.New("root")
	got := executeTemplate(tmpl, "no_such_template", nil)
	if !strings.Contains(string(got), "Error rendering") {
		t.Fatalf("expected fallback HTML, got %q", got)
	}
}

func TestIndexPageSearch(t *testing.T) {
	db := testStore(t)
	_ = db.UpsertRepo("hrodrig/gghstats", "main repo", 10, 0, 10, 0, 0, false, false, "")
	_ = db.UpsertRepo("hrodrig/pgwd", "other repo", 10, 0, 10, 0, 0, false, false, "")
	handler := New(Config{Store: db})

	req := httptest.NewRequest("GET", "/?q=gghstats", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "hrodrig/gghstats") {
		t.Fatalf("expected matching repo in response")
	}
	if strings.Contains(body, "hrodrig/pgwd") {
		t.Fatalf("did not expect non-matching repo in response")
	}
}

func TestRepoPage(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("owner/repo", "desc", 5, 1, 5, 0, 0, false, false, "")
	db.UpsertView("owner/repo", "2026-03-20", 10, 5)
	db.UpsertClone("owner/repo", "2026-03-20", 3, 2)

	handler := New(Config{Store: db})

	req := httptest.NewRequest("GET", "/owner/repo", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestRepoPageNotFound(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db})

	req := httptest.NewRequest("GET", "/nonexistent/repo", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "404") || !strings.Contains(body, "Repository not found") {
		t.Fatalf("expected brutalist 404 HTML, got %d bytes", len(body))
	}
	if !strings.Contains(body, "/nonexistent/repo") {
		t.Fatalf("expected path in body: %q", body)
	}
}

func TestUnknownAPIPathJSON404(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db})

	req := httptest.NewRequest("GET", "/api/unknown-endpoint", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want JSON", ct)
	}
	if !strings.Contains(w.Body.String(), `"error":"not_found"`) {
		t.Fatalf("body = %q", w.Body.String())
	}
}

func TestAPIWithoutToken(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db, APIToken: ""})

	req := httptest.NewRequest("GET", "/api/repos", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("status = %d, want 404 (API disabled)", w.Code)
	}
}

func TestAPIUnauthorized(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db, APIToken: "secret"})

	req := httptest.NewRequest("GET", "/api/repos", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAPIAuthorized(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "test", 10, 2, 10, 1, 0, false, false, "")
	db.UpsertView("a/b", "2026-03-20", 50, 20)

	handler := New(Config{Store: db, APIToken: "secret"})

	req := httptest.NewRequest("GET", "/api/repos", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["total_count"].(float64) != 1 {
		t.Errorf("total_count = %v, want 1", resp["total_count"])
	}
}
