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

func TestBadgeViewsAndStarsMetrics(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 7, 0, 0, 0, 0, false, false, "")
	db.UpsertView("a/b", time.Now().UTC().Format("2006-01-02"), 100, 10)

	handler := New(Config{Store: db, BadgePublic: true})

	for _, metric := range []string{"views", "stars"} {
		t.Run(metric, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/badge/a/b?metric="+metric, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != 200 {
				t.Fatalf("status = %d", w.Code)
			}
			body := w.Body.String()
			if metric == "views" && !strings.Contains(body, "100") {
				t.Fatalf("body = %q", body)
			}
			if metric == "stars" && !strings.Contains(body, ">7<") {
				t.Fatalf("body = %q", body)
			}
		})
	}
}

func TestBadgeLargeCloneCountUsesThousandsSeparator(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	db.UpsertClone("a/b", time.Now().UTC().Format("2006-01-02"), 1485, 1)

	handler := New(Config{Store: db, BadgePublic: true})
	req := httptest.NewRequest("GET", "/api/v1/badge/a/b", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "1,485") {
		t.Fatalf("body = %q, want thousands separator", w.Body.String())
	}
}

func TestBadgeFlatSquareStyle(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 1, 0, 0, 0, 0, false, false, "")
	handler := New(Config{Store: db, BadgePublic: true})

	req := httptest.NewRequest("GET", "/api/v1/badge/a/b?style=flat-square", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 || !strings.Contains(w.Body.String(), `rx="0"`) {
		t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
	}
}

func TestBadgeInvalidStyle(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	handler := New(Config{Store: db, BadgePublic: true})

	req := httptest.NewRequest("GET", "/api/v1/badge/a/b?style=round", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestBadgeNotFoundJSON(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db, BadgePublic: true})

	req := httptest.NewRequest("GET", "/api/v1/badge/x/y", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 404 || !strings.Contains(w.Body.String(), "not_found") {
		t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
	}
}

func TestBadgePrivateDisabledWithoutAPIToken(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db, BadgePublic: false, APIToken: ""})

	req := httptest.NewRequest("GET", "/api/v1/badge/a/b", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestBadgeCustomCacheMaxAge(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	handler := New(Config{Store: db, BadgePublic: true, BadgeCacheMaxAge: 60})

	req := httptest.NewRequest("GET", "/api/v1/badge/a/b", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !strings.Contains(w.Header().Get("Cache-Control"), "max-age=60") {
		t.Errorf("Cache-Control = %q", w.Header().Get("Cache-Control"))
	}
}

func TestPublicBaseURLForwardedProto(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "x", 1, 0, 1, 0, 0, false, false, "")

	req := httptest.NewRequest("GET", "/a/b", nil)
	req.Host = "internal:8080"
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	New(Config{Store: db}).ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), `data-base-url="https://internal:8080"`) {
		t.Error("expected https base URL from X-Forwarded-Proto")
	}
}

func TestFormatBadgeNumber(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{42, "42"},
		{999, "999"},
		{1000, "1,000"},
		{1485, "1,485"},
		{1234567, "1,234,567"},
	}
	for _, tt := range tests {
		if got := formatBadgeNumber(tt.n); got != tt.want {
			t.Errorf("formatBadgeNumber(%d) = %q, want %q", tt.n, got, tt.want)
		}
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
