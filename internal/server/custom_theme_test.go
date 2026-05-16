package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestResolveCustomCSS_Empty(t *testing.T) {
	p, q := ResolveCustomCSS("")
	if p != "" || q != "" {
		t.Fatalf("want empty, got path=%q q=%q", p, q)
	}
}

func TestResolveCustomCSS_RegularFile(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "theme.css")
	if err := os.WriteFile(fp, []byte("body{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, q := ResolveCustomCSS(fp)
	if p == "" || q == "" {
		t.Fatalf("want non-empty path and query, got path=%q q=%q", p, q)
	}
	st, err := os.Stat(p)
	if err != nil || !st.Mode().IsRegular() {
		t.Fatalf("resolved path not regular: %v", err)
	}
	if want := "v=" + strconv.FormatInt(st.ModTime().Unix(), 10); q != want {
		t.Errorf("query = %q, want %q", q, want)
	}
}

func TestResolveCustomCSS_NotRegular(t *testing.T) {
	dir := t.TempDir()
	p, q := ResolveCustomCSS(dir)
	if p != "" || q != "" {
		t.Fatalf("directory must not resolve, got path=%q q=%q", p, q)
	}
}

func TestHandleCustomCSS(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "t.css")
	content := "/* ok */\nbody.app-brutalist{--brutal-accent:red;}\n"
	if err := os.WriteFile(fp, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	h := handleCustomCSS(fp)
	req := httptest.NewRequest(http.MethodGet, "/theme/custom.css", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/css; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	if w.Body.String() != content {
		t.Errorf("body mismatch")
	}
}
