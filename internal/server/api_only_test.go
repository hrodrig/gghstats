package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIOnlySkipsHTMLAndSEO(t *testing.T) {
	db := testStore(t)
	h := New(Config{Store: db, APIToken: "secret", APIOnly: true, DisableMetrics: true})

	for _, path := range []string{"/", "/h2h", "/robots.txt", "/sitemap.xml"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", path, w.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, HealthzPath, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("healthz status = %d", w.Code)
	}
}

func TestAPIOnlyKeepsJSONAPI(t *testing.T) {
	db := testStore(t)
	if err := db.UpsertRepo("o/r", "o/r", 1, 0, 1, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	h := New(Config{Store: db, APIToken: "tok", APIOnly: true, DisableMetrics: true})
	req := httptest.NewRequest(http.MethodGet, "/api/repos", nil)
	req.Header.Set("x-api-token", "tok")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
}
