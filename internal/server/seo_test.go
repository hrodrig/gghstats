package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestHostName(t *testing.T) {
	tests := []struct {
		hostport string
		want     string
	}{
		{"", ""},
		{"example.com", "example.com"},
		{"example.com:443", "example.com"},
		{"127.0.0.1:8080", "127.0.0.1"},
		{"[::1]:8080", "[::1]"},
		{"[2001:db8::1]:443", "[2001:db8::1]"},
		{"[::1]", "[::1]"},
		{"[incomplete", "[incomplete"},
	}
	for _, tt := range tests {
		if got := requestHostName(tt.hostport); got != tt.want {
			t.Errorf("requestHostName(%q) = %q, want %q", tt.hostport, got, tt.want)
		}
	}
}

func TestSEOIndexable(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		publicURL string
		want      bool
	}{
		{"public url overrides localhost", "127.0.0.1:8080", "https://stats.example.com", true},
		{"localhost", "localhost:8080", "", false},
		{"127.0.0.1", "127.0.0.1:8080", "", false},
		{"127.x block", "127.0.0.2:8080", "", false},
		{"ipv6 loopback", "[::1]:8080", "", false},
		{"production host", "gghstats.example.com:443", "", true},
		{"empty host", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Host = tt.host
			if got := seoIndexable(req, tt.publicURL); got != tt.want {
				t.Fatalf("seoIndexable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRobotsLocalhostDisallow(t *testing.T) {
	db := testStore(t)
	h := New(Config{Store: db})
	req := httptest.NewRequest("GET", "/robots.txt", nil)
	req.Host = "127.0.0.1:8080"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Disallow: /") {
		t.Fatalf("body = %q", body)
	}
	if strings.Contains(body, "Sitemap:") {
		t.Fatalf("unexpected sitemap on localhost: %q", body)
	}
}

func TestRobotsPublicURL(t *testing.T) {
	db := testStore(t)
	h := New(Config{Store: db, PublicURL: "https://stats.example.com"})
	req := httptest.NewRequest("GET", "/robots.txt", nil)
	req.Host = "internal:8080"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "Allow: /") {
		t.Fatalf("body = %q", body)
	}
	if !strings.Contains(body, "Sitemap: https://stats.example.com/sitemap.xml") {
		t.Fatalf("body = %q", body)
	}
	if !strings.Contains(body, "Disallow: /api/") {
		t.Fatalf("body = %q", body)
	}
}

func TestSitemapIncludesRepos(t *testing.T) {
	db := testStore(t)
	if err := db.UpsertRepo("acme/widget", "x", 1, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	h := New(Config{Store: db, PublicURL: "https://stats.example.com"})
	req := httptest.NewRequest("GET", "/sitemap.xml", nil)
	req.Host = "stats.example.com"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		"<loc>https://stats.example.com/</loc>",
		"<loc>https://stats.example.com/h2h</loc>",
		"<loc>https://stats.example.com/acme/widget</loc>",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in body:\n%s", want, body)
		}
	}
}

func TestSitemapLocalhostEmpty(t *testing.T) {
	db := testStore(t)
	h := New(Config{Store: db})
	req := httptest.NewRequest("GET", "/sitemap.xml", nil)
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	body := w.Body.String()
	if strings.Contains(body, "<loc>http://localhost:8080/") {
		t.Fatalf("unexpected localhost URL in sitemap: %q", body)
	}
}

func TestRobotsProductionHostWithoutPublicURL(t *testing.T) {
	db := testStore(t)
	h := New(Config{Store: db})
	req := httptest.NewRequest("GET", "/robots.txt", nil)
	req.Host = "gghstats.example.com"
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "Sitemap: https://gghstats.example.com/sitemap.xml") {
		t.Fatalf("body = %q", body)
	}
}

func TestSitemapProductionHostWithoutPublicURL(t *testing.T) {
	db := testStore(t)
	h := New(Config{Store: db})
	req := httptest.NewRequest("GET", "/sitemap.xml", nil)
	req.Host = "stats.example.com:8443"
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "<loc>https://stats.example.com:8443/</loc>") {
		t.Fatalf("body = %q", body)
	}
}

func TestSitemapNilStore(t *testing.T) {
	h := New(Config{PublicURL: "https://stats.example.com"})
	req := httptest.NewRequest("GET", "/sitemap.xml", nil)
	req.Host = "stats.example.com"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "<loc>https://stats.example.com/h2h</loc>") {
		t.Fatalf("body = %q", body)
	}
	if strings.Contains(body, "/acme/") {
		t.Fatalf("unexpected repo URL with nil store: %q", body)
	}
}

func TestSitemapListReposErrorStillServesCoreURLs(t *testing.T) {
	db := testStore(t)
	db.Close()
	h := New(Config{Store: db, PublicURL: "https://stats.example.com"})
	req := httptest.NewRequest("GET", "/sitemap.xml", nil)
	req.Host = "stats.example.com"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	body := w.Body.String()
	for _, want := range []string{
		"<loc>https://stats.example.com/</loc>",
		"<loc>https://stats.example.com/h2h</loc>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in body:\n%s", want, body)
		}
	}
}

func TestSitemapSkipsInvalidRepoNames(t *testing.T) {
	db := testStore(t)
	for _, name := range []string{"acme/ok", "no-slash", "a/b/c", "has space/repo"} {
		if err := db.UpsertRepo(name, "", 0, 0, 0, 0, 0, false, false, ""); err != nil {
			t.Fatal(err)
		}
	}
	h := New(Config{Store: db, PublicURL: "https://stats.example.com"})
	req := httptest.NewRequest("GET", "/sitemap.xml", nil)
	req.Host = "stats.example.com"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "<loc>https://stats.example.com/acme/ok</loc>") {
		t.Fatalf("body = %q", body)
	}
	if strings.Contains(body, "no-slash") || strings.Contains(body, "a/b/c") {
		t.Fatalf("unexpected invalid repo in sitemap:\n%s", body)
	}
	if strings.Contains(body, "has space/repo") {
		t.Fatalf("repo name with space should be skipped:\n%s", body)
	}
}

func TestRepoPageLoc(t *testing.T) {
	loc, ok := repoPageLoc("https://example.com", "o/r")
	if !ok || loc != "https://example.com/o/r" {
		t.Fatalf("got %q ok=%v", loc, ok)
	}
	cases := []struct {
		name string
		ok   bool
	}{
		{"bad", false},
		{"", false},
		{" ", false},
		{"owner/", false},
		{"/repo", false},
		{"a/b/c", false},
		{"has space/repo", false},
	}
	for _, c := range cases {
		if _, ok := repoPageLoc("https://example.com", c.name); ok != c.ok {
			t.Errorf("repoPageLoc(%q) ok = %v, want %v", c.name, ok, c.ok)
		}
	}
}
