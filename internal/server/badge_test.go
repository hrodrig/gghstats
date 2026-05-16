package server

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBadgeSVGTotalClones(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "x", 5, 0, 5, 0, 0, false, false, "")
	db.UpsertClone("a/b", time.Now().UTC().Format("2006-01-02"), 813, 10)

	handler := New(Config{Store: db, BadgePublic: true})

	req := httptest.NewRequest("GET", "/api/v1/badge/a/b?metric=clones", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(w.Header().Get("Content-Type"), "image/svg+xml") {
		t.Errorf("Content-Type = %q", w.Header().Get("Content-Type"))
	}
	if !strings.Contains(body, "813") || !strings.Contains(body, ">clones<") {
		t.Fatalf("svg body missing expected content: %q", body)
	}
	if cc := w.Header().Get("Cache-Control"); !strings.Contains(cc, "max-age=300") {
		t.Errorf("Cache-Control = %q, want max-age=300", cc)
	}
}

func TestBadgeMetricClones30d(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	today := time.Now().UTC().Format("2006-01-02")
	db.UpsertClone("a/b", today, 42, 5)

	handler := New(Config{Store: db, BadgePublic: true})
	req := httptest.NewRequest("GET", "/api/v1/badge/a/b?metric=clones_30d", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "42") {
		t.Fatalf("body = %q", w.Body.String())
	}
}

func TestBadgeNotFound(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db, BadgePublic: true})

	req := httptest.NewRequest("GET", "/api/v1/badge/none/here", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if !strings.Contains(w.Body.String(), "unknown") {
		t.Fatalf("body = %q", w.Body.String())
	}
}

func TestBadgeInvalidMetricJSON(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	handler := New(Config{Store: db, BadgePublic: true})

	req := httptest.NewRequest("GET", "/api/v1/badge/a/b?metric=nope", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid metric") {
		t.Fatalf("body = %q", w.Body.String())
	}
}

func TestBadgeRequiresTokenWhenNotPublic(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	handler := New(Config{Store: db, BadgePublic: false, APIToken: "secret"})

	req := httptest.NewRequest("GET", "/api/v1/badge/a/b", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestBadgePublicWithoutAPIToken(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	handler := New(Config{Store: db, BadgePublic: true, APIToken: ""})

	req := httptest.NewRequest("GET", "/api/v1/badge/a/b?metric=stars", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestBadgeSVGAlias(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 3, 0, 3, 0, 0, false, false, "")
	handler := New(Config{Store: db, BadgePublic: true})

	req := httptest.NewRequest("GET", "/api/v1/badge/a/b.svg?metric=stars", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), ">3<") {
		t.Fatalf("body = %q", w.Body.String())
	}
}

func TestRepoPageBadgeEmbed(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "test", 10, 2, 10, 1, 0, false, false, "")

	handler := New(Config{Store: db, PublicURL: "https://stats.example.com"})
	req := httptest.NewRequest("GET", "/a/b", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `id="gghstats-badge-embed"`) {
		t.Error("expected badge embed block")
	}
	if !strings.Contains(body, `data-base-url="https://stats.example.com"`) {
		t.Error("expected public base URL in embed block")
	}
	if !strings.Contains(body, `id="badge-metric"`) {
		t.Error("expected metric selector")
	}
}
