package demo

import (
	"path/filepath"
	"testing"

	"github.com/hrodrig/gghstats/internal/store"
)

func TestSeedIfEmpty(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "demo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	if err := SeedIfEmpty(s); err != nil {
		t.Fatal(err)
	}
	n, err := s.RepoCount()
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("repos = %d, want 3", n)
	}
	// Second call must not duplicate.
	if err := SeedIfEmpty(s); err != nil {
		t.Fatal(err)
	}
	n2, err := s.RepoCount()
	if err != nil {
		t.Fatal(err)
	}
	if n2 != 3 {
		t.Fatalf("after re-seed repos = %d, want 3", n2)
	}

	sum, err := s.RepoByName("demo/alpha")
	if err != nil || sum == nil {
		t.Fatalf("alpha missing: %v", err)
	}
	if sum.TotalClones < 1 {
		t.Fatalf("expected clone totals after deltas, got %d", sum.TotalClones)
	}
}
