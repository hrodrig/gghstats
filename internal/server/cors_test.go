package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseCORSOrigins(t *testing.T) {
	if got := ParseCORSOrigins(""); got != nil {
		t.Fatalf("empty = %v", got)
	}
	got := ParseCORSOrigins(" https://a.example , https://b.example ")
	if len(got) != 2 || got[0] != "https://a.example" || got[1] != "https://b.example" {
		t.Fatalf("got %#v", got)
	}
}

func TestSetAPICORSOpen(t *testing.T) {
	w := httptest.NewRecorder()
	setAPICORS(w, httptest.NewRequest(http.MethodGet, "/", nil), nil)
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal(w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestSetAPICORSAllowList(t *testing.T) {
	origins := []string{"https://app.example"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://app.example")
	w := httptest.NewRecorder()
	setAPICORS(w, req, origins)
	if w.Header().Get("Access-Control-Allow-Origin") != "https://app.example" {
		t.Fatal(w.Header().Get("Access-Control-Allow-Origin"))
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("Origin", "https://evil.example")
	w2 := httptest.NewRecorder()
	setAPICORS(w2, req2, origins)
	if w2.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("should not echo disallowed origin")
	}
}

func TestCORSIsOpen(t *testing.T) {
	if !CORSIsOpen(nil) {
		t.Fatal("nil should be open")
	}
	if !CORSIsOpen([]string{"*"}) {
		t.Fatal("* should be open")
	}
	if CORSIsOpen([]string{"https://x"}) {
		t.Fatal("specific should not be open")
	}
}
