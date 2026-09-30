package demo

import (
	"path/filepath"
	"testing"
	"time"

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
	if n != 4 {
		t.Fatalf("repos = %d, want 4", n)
	}
	// Second call must not duplicate.
	if err := SeedIfEmpty(s); err != nil {
		t.Fatal(err)
	}
	n2, err := s.RepoCount()
	if err != nil {
		t.Fatal(err)
	}
	if n2 != 4 {
		t.Fatalf("after re-seed repos = %d, want 4", n2)
	}

	sum, err := s.RepoByName("demo/alpha")
	if err != nil || sum == nil {
		t.Fatalf("alpha missing: %v", err)
	}
	if sum.TotalClones < 1 {
		t.Fatalf("expected clone totals after deltas, got %d", sum.TotalClones)
	}

	sparse, err := s.RepoByName("demo/sparse-stars")
	if err != nil || sparse == nil {
		t.Fatalf("sparse-stars missing: %v", err)
	}
	if sparse.Stars != 22 {
		t.Fatalf("sparse-stars KPI = %d, want 22", sparse.Stars)
	}
	hist, err := s.StarsByRepo("demo/sparse-stars")
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 7 || hist[len(hist)-1].Total != 11 {
		t.Fatalf("sparse history want 7 rows ending at 11, got %+v", hist)
	}
}

func TestApplyUpstreamStaleFreeze(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "demo-stale.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err := Seed(s); err != nil {
		t.Fatal(err)
	}
	if err := ApplyUpstreamStaleFreeze(s); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	got, err := store.DetectFleetUpstreamStale(s, store.ReportVisibility{}, 3, now, true)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Active || got.Since == "" || got.DaysStuck < 1 {
		t.Fatalf("want active fleet stale, got %+v", got)
	}
}

func TestApplyUpstreamStaleFreeze_NilAndEmpty(t *testing.T) {
	if err := ApplyUpstreamStaleFreeze(nil); err == nil {
		t.Fatal("nil store")
	}
	s, err := store.Open(filepath.Join(t.TempDir(), "empty.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err := ApplyUpstreamStaleFreeze(s); err == nil {
		t.Fatal("empty repos")
	}
}
