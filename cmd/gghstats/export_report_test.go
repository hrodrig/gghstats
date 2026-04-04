package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/store"
)

// stdoutSwapMu serializes tests that replace os.Stdout (parallel subtests elsewhere in this package).
var stdoutSwapMu sync.Mutex

func TestRunExportStdout(t *testing.T) {
	stdoutSwapMu.Lock()
	defer stdoutSwapMu.Unlock()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "e.db")

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	day := time.Now().UTC().Format("2006-01-02")
	if err := s.UpsertRepo("my/repo", "desc", 1, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertView("my/repo", day, 10, 5); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertClone("my/repo", day, 3, 2); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GGHSTATS_REPO", "my/repo")
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "tok")

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	errCh := make(chan error, 1)
	go func() { errCh <- runExport([]string{"-db", dbPath}); w.Close() }()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stdout = old
	_ = r.Close()

	if err := <-errCh; err != nil {
		t.Fatalf("runExport: %v", err)
	}
	out := buf.String()
	if len(out) < 20 || !bytes.Contains([]byte(out), []byte("# Views")) {
		t.Fatalf("unexpected CSV output: %q", out)
	}
}

func TestRunReportStdout(t *testing.T) {
	stdoutSwapMu.Lock()
	defer stdoutSwapMu.Unlock()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "r.db")

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	day := time.Now().UTC().Format("2006-01-02")
	if err := s.UpsertRepo("my/repo", "desc", 1, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertView("my/repo", day, 10, 5); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GGHSTATS_REPO", "my/repo")
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "tok")

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	errCh := make(chan error, 1)
	go func() { errCh <- runReport([]string{"-db", dbPath}); w.Close() }()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stdout = old
	_ = r.Close()

	if err := <-errCh; err != nil {
		t.Fatalf("runReport: %v", err)
	}
	if buf.Len() < 10 {
		t.Fatalf("expected terminal output, got %q", buf.String())
	}
}
