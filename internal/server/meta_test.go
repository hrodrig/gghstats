package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPageCanonicalURL_IndexStripsNoise(t *testing.T) {
	base := "https://stats.example.com"
	r := httptest.NewRequest("GET", "/?lang=es&page=2&sort=stars&dir=desc&per_page=25", nil)
	got := pageCanonicalURL(base, r, "index")
	if got != "https://stats.example.com/" {
		t.Fatalf("got %q", got)
	}
}

func TestPageCanonicalURL_IndexKeepsSearch(t *testing.T) {
	base := "https://stats.example.com"
	r := httptest.NewRequest("GET", "/?q=myapp&lang=de", nil)
	got := pageCanonicalURL(base, r, "index")
	if got != "https://stats.example.com/?q=myapp" {
		t.Fatalf("got %q", got)
	}
}

func TestPageCanonicalURL_H2H(t *testing.T) {
	base := "https://gghstats.example.com"
	r := httptest.NewRequest("GET", "/h2h?a=o/r1&b=o/r2&w=30d&lang=fr", nil)
	got := pageCanonicalURL(base, r, "h2h")
	want := "https://gghstats.example.com/h2h?a=o%2Fr1&b=o%2Fr2&w=30d"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPageCanonicalURL_Repo(t *testing.T) {
	base := "https://gghstats.example.com"
	r := httptest.NewRequest("GET", "/hrodrig/gghstats?lang=es", nil)
	r.SetPathValue("owner", "hrodrig")
	r.SetPathValue("repo", "gghstats")
	got := pageCanonicalURL(base, r, "repo")
	if got != "https://gghstats.example.com/hrodrig/gghstats" {
		t.Fatalf("got %q", got)
	}
}

func TestIndexPageHasCanonicalAndDescription(t *testing.T) {
	db := testStore(t)
	h := New(Config{Store: db, PublicURL: "https://stats.example.com"})
	req := httptest.NewRequest("GET", "/?lang=es&page=2&sort=name", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, `rel="canonical" href="https://stats.example.com/"`) {
		t.Fatalf("missing index canonical in body")
	}
	if !strings.Contains(body, `name="description"`) {
		t.Fatalf("missing meta description")
	}
	if strings.Contains(body, `lang=es`) && strings.Contains(body, `rel="canonical" href="https://stats.example.com/?`) {
		t.Fatalf("canonical should not include lang")
	}
}

func TestNotFoundHasNoindex(t *testing.T) {
	db := testStore(t)
	h := New(Config{Store: db, PublicURL: "https://stats.example.com"})
	req := httptest.NewRequest("GET", "/missing/path", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `name="robots" content="noindex, nofollow"`) {
		t.Fatalf("missing noindex on 404")
	}
}
