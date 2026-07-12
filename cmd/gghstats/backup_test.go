package main

import (
	"path/filepath"
	"testing"

	"github.com/hrodrig/gghstats/internal/store"
)

func TestRunBackupRestoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "data.db")
	bakPath := filepath.Join(dir, "backup.db")
	outPath := filepath.Join(dir, "out.db")

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertRepo("a/b", "x", 2, 0, 2, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	s.Close()

	if err := runBackup([]string{"--db", dbPath, "--output", bakPath}); err != nil {
		t.Fatalf("runBackup: %v", err)
	}
	if err := runRestore([]string{"--input", bakPath, "--db", outPath}); err != nil {
		t.Fatalf("runRestore: %v", err)
	}

	out, err := store.Open(outPath)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	sum, err := out.RepoByName("a/b")
	if err != nil || sum == nil {
		t.Fatalf("repo missing after restore: %v %v", sum, err)
	}
}

func TestRunBackupMissingOutput(t *testing.T) {
	if err := runBackup(nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestRunRestoreMissingInput(t *testing.T) {
	if err := runRestore(nil); err == nil {
		t.Fatal("expected error")
	}
}
