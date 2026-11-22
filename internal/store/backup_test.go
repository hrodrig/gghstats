package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("backup mode=%o, want 0600", st.Mode().Perm())
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

func TestBackupToChmodFailureRemovesFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "live.db")
	bakPath := filepath.Join(dir, "snap.db")

	s, err := Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.UpsertRepo("o/r", "d", 1, 0, 1, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}

	orig := chmodBackup
	chmodBackup = func(string, os.FileMode) error {
		return errors.New("chmod denied")
	}
	t.Cleanup(func() { chmodBackup = orig })

	err = s.BackupTo(bakPath)
	if err == nil || !strings.Contains(err.Error(), "chmod backup") {
		t.Fatalf("err=%v, want chmod backup", err)
	}
	if _, err := os.Stat(bakPath); !os.IsNotExist(err) {
		t.Fatalf("backup file should be removed after chmod failure, stat err=%v", err)
	}
}

func TestQuoteSQLiteLiteral(t *testing.T) {
	if got := quoteSQLiteLiteral(`a'b`); got != `'a''b'` {
		t.Fatalf("got %q", got)
	}
}
