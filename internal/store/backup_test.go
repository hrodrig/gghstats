package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupToAndRestoreDBFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "live.db")
	bakPath := filepath.Join(dir, "snap.db")
	restoredPath := filepath.Join(dir, "restored.db")

	s, err := Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertRepo("o/r", "d", 1, 0, 1, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.BackupTo(bakPath); err != nil {
		t.Fatalf("BackupTo: %v", err)
	}
	if err := s.BackupTo(bakPath); err == nil {
		t.Fatal("expected error when destination exists")
	}
	s.Close()

	st, err := os.Stat(bakPath)
	if err != nil || st.Size() == 0 {
		t.Fatalf("backup missing or empty: %v", err)
	}

	if err := RestoreDBFile(bakPath, restoredPath); err != nil {
		t.Fatalf("RestoreDBFile: %v", err)
	}
	out, err := Open(restoredPath)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	sum, err := out.RepoByName("o/r")
	if err != nil || sum == nil {
		t.Fatalf("restored repo: sum=%v err=%v", sum, err)
	}
}

func TestQuoteSQLiteLiteral(t *testing.T) {
	if got := quoteSQLiteLiteral(`a'b`); got != `'a''b'` {
		t.Fatalf("got %q", got)
	}
}
