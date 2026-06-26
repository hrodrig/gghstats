package store

import (
	"path/filepath"
	"testing"
)

func TestOpenAppliesPRAGMAs(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "pragma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	var journalMode string
	if err := s.DB().QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want wal", journalMode)
	}

	// synchronous: 0=OFF, 1=NORMAL, 2=FULL, 3=EXTRA
	var sync int
	if err := s.DB().QueryRow("PRAGMA synchronous").Scan(&sync); err != nil {
		t.Fatal(err)
	}
	if sync != 1 {
		t.Errorf("synchronous = %d, want 1 (NORMAL)", sync)
	}

	var busyMs int
	if err := s.DB().QueryRow("PRAGMA busy_timeout").Scan(&busyMs); err != nil {
		t.Fatal(err)
	}
	if busyMs != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", busyMs)
	}
}

func TestOpenCapsConnectionPool(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "pool.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	stats := s.DB().Stats()
	if stats.MaxOpenConnections != 4 {
		t.Errorf("MaxOpenConnections = %d, want 4", stats.MaxOpenConnections)
	}
}
