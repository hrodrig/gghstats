package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRejectsDirectoryPath(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "dbdir")
	if err := os.Mkdir(dbPath, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Open(dbPath)
	if err == nil {
		t.Fatal("Open: expected error when path is an existing directory")
	}
}
