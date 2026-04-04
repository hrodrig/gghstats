package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunServeMissingToken(t *testing.T) {
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
	if err := runServe(nil); err == nil {
		t.Fatal("expected error when GGHSTATS_GITHUB_TOKEN is empty")
	}
}

func TestRunServeHelp(t *testing.T) {
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
	if err := runServe([]string{"-h"}); err != nil {
		t.Fatalf("help: %v", err)
	}
	if err := runServe([]string{"--help"}); err != nil {
		t.Fatalf("--help: %v", err)
	}
}

func TestRunServeUnknownFlag(t *testing.T) {
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "tok")
	err := runServe([]string{"-undefined"})
	if err == nil {
		t.Fatal("expected flag parse error")
	}
}

func TestRunServeOpenDatabaseFails(t *testing.T) {
	root := t.TempDir()
	dbDir := filepath.Join(root, "dbdir")
	if err := os.Mkdir(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "tok")
	t.Setenv("GGHSTATS_DB", dbDir)
	err := runServe(nil)
	if err == nil {
		t.Fatal("expected database open error")
	}
	if !strings.Contains(err.Error(), "database") {
		t.Fatalf("unexpected error: %v", err)
	}
}
