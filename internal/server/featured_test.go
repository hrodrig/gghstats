package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// R1: empty catalog → GET / has no href="/featured".
func TestFeaturedNavHiddenWhenEmpty(t *testing.T) {
	h := New(Config{Store: testStore(t)})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if strings.Contains(w.Body.String(), `href="/featured"`) {
		t.Error("R1: empty catalog must not render a /featured nav link")
	}
}

// R5: API-only → GET /featured is not an HTML dashboard page.
func TestFeaturedAPIOonlyNotFound(t *testing.T) {
	h := New(Config{Store: testStore(t), APIToken: "tok", APIOnly: true, DisableMetrics: true})
	req := httptest.NewRequest(http.MethodGet, "/featured", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("R5: API-only /featured status = %d, want 404", w.Code)
	}
}

// R9: after featured add, GET / has the link; after removing the last row,
// the link is gone again.
func TestFeaturedNavToggle(t *testing.T) {
	db := testStore(t)
	if err := db.AddFeatured("hrodrig/awesome-readme"); err != nil {
		t.Fatal(err)
	}
	h := New(Config{Store: db})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `href="/featured"`) {
		t.Error("R9: after featured add, GET / must render a /featured nav link")
	}

	if _, err := db.RemoveFeatured("hrodrig/awesome-readme"); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if strings.Contains(w.Body.String(), `href="/featured"`) {
		t.Error("R9: after removing the last featured row, the /featured nav link must disappear")
	}
}

// The /featured page itself: 200 and renders the fork card (upstream link).
func TestFeaturedPageShell(t *testing.T) {
	db := testStore(t)
	if err := db.AddFeatured("hrodrig/awesome-readme"); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertFeaturedMeta(
		"hrodrig/awesome-readme",
		"matiassingers/awesome-readme",
		"matiassingers/awesome-readme",
		"A curated list of awesome readmes",
		15000,
		true,
	); err != nil {
		t.Fatal(err)
	}
	h := New(Config{Store: db})
	req := httptest.NewRequest(http.MethodGet, "/featured", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "matiassingers/awesome-readme") {
		t.Error("expected upstream repo name in featured page")
	}
	if !strings.Contains(body, "15,000") {
		t.Error("expected formatted upstream stars (thousands separator) in featured page")
	}
	// The card title/link must be an external GitHub URL, not an internal route
	// (https://github.com/{upstream_full_name}). Regression: an earlier draft
	// linked to href="/{name}", which 404s on the dashboard.
	if !strings.Contains(body, `href="https://github.com/matiassingers/awesome-readme"`) {
		t.Error("expected card title/link to point to https://github.com/<upstream>")
	}
	if strings.Contains(body, `href="/matiassingers/awesome-readme"`) {
		t.Error("card title must not link to an internal route")
	}
}

// The /featured page renders a friendly empty state when the showcase is empty.
func TestFeaturedPageEmpty(t *testing.T) {
	h := New(Config{Store: testStore(t)})
	req := httptest.NewRequest(http.MethodGet, "/featured", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `data-gghstats-role="featured-empty"`) {
		t.Errorf("expected empty-state marker in featured page body")
	}
	if !strings.Contains(body, `data-gghstats-role="featured-empty-cmd"`) {
		t.Errorf("expected featured-add command on true empty showcase")
	}
}

// #75: filtered Featured search with zero matches must not reuse featured-add hint.
func TestFeaturedPageSearchEmpty(t *testing.T) {
	db := testStore(t)
	if err := db.AddFeatured("hrodrig/awesome-readme"); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertFeaturedMeta(
		"hrodrig/awesome-readme",
		"matiassingers/awesome-readme",
		"matiassingers/awesome-readme",
		"A curated list of awesome readmes",
		15000,
		true,
	); err != nil {
		t.Fatal(err)
	}
	h := New(Config{Store: db})
	req := httptest.NewRequest(http.MethodGet, "/featured?q=sfdsdf", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `data-gghstats-role="featured-search-empty"`) {
		t.Fatal("expected featured search-empty marker")
	}
	if strings.Contains(body, `data-gghstats-role="featured-empty-cmd"`) {
		t.Fatal("filter miss must not show featured-add command")
	}
	if strings.Contains(body, "No featured repositories yet") {
		t.Fatal("filter miss must not reuse empty-showcase title")
	}
	if !strings.Contains(body, "No featured repositories match this search") {
		t.Fatal("expected featured search-empty title copy")
	}
}

// With GGHSTATS_COMPACT_NUMBERS (CompactNumbers=true), large upstream stars
// render in compact metric notation (e.g. 15.0k) rather than 15,000.
func TestFeaturedPageCompactNumbers(t *testing.T) {
	db := testStore(t)
	if err := db.AddFeatured("hrodrig/awesome-readme"); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertFeaturedMeta(
		"hrodrig/awesome-readme",
		"matiassingers/awesome-readme",
		"matiassingers/awesome-readme",
		"A curated list of awesome readmes",
		15000,
		true,
	); err != nil {
		t.Fatal(err)
	}
	h := New(Config{Store: db, CompactNumbers: true})
	req := httptest.NewRequest(http.MethodGet, "/featured", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "15.0k") {
		t.Errorf("expected compact stars (15.0k) in featured page, got body without it")
	}
	if strings.Contains(body, "15,000") {
		t.Errorf("compact mode must not render thousands separators (15,000)")
	}
}

// With enough entries, /featured paginates: query params page/per_page select a
// slice, and the page renders search + sort + pagination controls.
func TestFeaturedPagePagination(t *testing.T) {
	db := testStore(t)
	for i := 0; i < 30; i++ {
		name := fmt.Sprintf("owner/repo-%02d", i)
		if err := db.AddFeatured(name); err != nil {
			t.Fatal(err)
		}
		if err := db.UpsertFeaturedMeta(name, "", name, "desc", i, false); err != nil {
			t.Fatal(err)
		}
	}

	h := New(Config{Store: db})

	// page 1 default (per_page=25) → shows first 25, total 30
	req := httptest.NewRequest(http.MethodGet, "/featured", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `data-gghstats-role="featured-sort-toolbar"`) {
		t.Error("expected sort toolbar in featured page")
	}
	if !strings.Contains(body, "Showing 1–25 of 30") {
		t.Errorf("expected showing line 1–25 of 30, got: %s", extractShowing(body))
	}
	if !strings.Contains(body, "owner/repo-00") {
		t.Error("page 1 should include owner/repo-00")
	}

	// page 2 (per_page=25) → items 26–30 only
	req = httptest.NewRequest(http.MethodGet, "/featured?page=2&per_page=25", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body = w.Body.String()
	if !strings.Contains(body, "Showing 26–30 of 30") {
		t.Errorf("expected showing line 26–30 of 30, got: %s", extractShowing(body))
	}
	if strings.Contains(body, "owner/repo-00") {
		t.Error("page 2 must not include owner/repo-00")
	}
	if !strings.Contains(body, "owner/repo-25") {
		t.Error("page 2 should include owner/repo-25")
	}

	// search filters the catalog before pagination
	req = httptest.NewRequest(http.MethodGet, "/featured?q=repo-29", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body = w.Body.String()
	if !strings.Contains(body, "owner/repo-29") {
		t.Error("search should surface owner/repo-29")
	}
	if strings.Contains(body, "owner/repo-00") {
		t.Error("search narrowed to repo-29 must not include owner/repo-00")
	}
}

func extractShowing(body string) string {
	i := strings.Index(body, "Showing ")
	if i < 0 {
		return ""
	}
	j := strings.Index(body[i:], "</p>")
	if j < 0 {
		return body[i:]
	}
	return body[i : i+j]
}

func TestFeaturedListsCatalogWithoutCollectedRepo(t *testing.T) {
	db := testStore(t)
	if err := db.AddFeatured("avelino/awesome-go"); err != nil {
		t.Fatal(err)
	}
	h := New(Config{Store: db, APIToken: "tok", DisableMetrics: true})

	index := httptest.NewRecorder()
	h.ServeHTTP(index, httptest.NewRequest(http.MethodGet, "/", nil))
	if !strings.Contains(index.Body.String(), `href="/featured"`) {
		t.Fatal("nav must show Featured when catalog has a vitrine-only row")
	}

	page := httptest.NewRecorder()
	h.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/featured", nil))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "avelino/awesome-go") {
		t.Fatalf("featured page missing vitrine-only entry: status=%d body=%s", page.Code, page.Body.String())
	}
}
