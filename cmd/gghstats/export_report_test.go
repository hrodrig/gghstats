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

func TestRunExportToFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "e2.db")
	outPath := filepath.Join(dir, "out.csv")

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	day := time.Now().UTC().Format("2006-01-02")
	if err := s.UpsertRepo("my/repo", "d", 1, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertView("my/repo", day, 1, 1); err != nil {
		t.Fatal(err)
	}

	err = runExport([]string{
		"-repo", "my/repo", "-token", "tok", "-db", dbPath,
		"-days", "7",
		"-output", outPath,
	})
	if err != nil {
		t.Fatalf("runExport: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 10 || !bytes.Contains(data, []byte("# Views")) {
		t.Fatalf("unexpected file: %q", data)
	}
}

func TestRunExportCreateFileError(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "e3.db")
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.UpsertRepo("my/repo", "d", 1, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertView("my/repo", time.Now().UTC().Format("2006-01-02"), 1, 1); err != nil {
		t.Fatal(err)
	}

	badPath := filepath.Join(dir, "nope", "sub", "out.csv")
	err = runExport([]string{
		"-repo", "my/repo", "-token", "tok", "-db", dbPath,
		"-output", badPath,
	})
	if err == nil {
		t.Fatal("expected error creating output under missing directory")
	}
}

func TestRunReportWithDaysFlag(t *testing.T) {
	stdoutSwapMu.Lock()
	defer stdoutSwapMu.Unlock()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "r2.db")

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	if err := s.UpsertRepo("my/repo", "desc", 1, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertView("my/repo", time.Now().UTC().Format("2006-01-02"), 1, 1); err != nil {
		t.Fatal(err)
	}

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	errCh := make(chan error, 1)
	go func() {
		errCh <- runReport([]string{"-repo", "my/repo", "-token", "tok", "-db", dbPath, "-days", "1"})
		w.Close()
	}()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stdout = old
	_ = r.Close()

	if err := <-errCh; err != nil {
		t.Fatalf("runReport: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("Traffic report")) {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

func TestRunReportUnknownFlag(t *testing.T) {
	err := runReport([]string{"-undefined-flag"})
	if err == nil {
		t.Fatal("expected flag parse error")
	}
}

func TestRunExportUnknownFlag(t *testing.T) {
	err := runExport([]string{"-undefined-flag"})
	if err == nil {
		t.Fatal("expected flag parse error")
	}
}

func TestRunReportMissingToken(t *testing.T) {
	t.Setenv("GGHSTATS_REPO", "o/r")
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
	err := runReport(nil)
	if err == nil {
		t.Fatal("expected missing token error")
	}
}

func TestRunReportOpenDBFails(t *testing.T) {
	root := t.TempDir()
	dbDir := filepath.Join(root, "dbdir")
	if err := os.Mkdir(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GGHSTATS_REPO", "o/r")
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "tok")
	err := runReport([]string{"-db", dbDir})
	if err == nil {
		t.Fatal("expected database open error")
	}
}

func TestRunExportOpenDBFails(t *testing.T) {
	root := t.TempDir()
	dbDir := filepath.Join(root, "dbdir")
	if err := os.Mkdir(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GGHSTATS_REPO", "o/r")
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "tok")
	err := runExport([]string{"-db", dbDir})
	if err == nil {
		t.Fatal("expected database open error")
	}
}
