package h2h

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/store"
)

func TestLoadRepoMetrics(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "h2h-metrics.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	today := time.Now().UTC().Format("2006-01-02")
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	weekAgo := time.Now().UTC().AddDate(0, 0, -8).Format("2006-01-02")

	if err := s.UpsertRepo("o/r", "desc", 3, 0, 3, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertClone("o/r", today, 10, 5); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertClone("o/r", yesterday, 4, 2); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertClone("o/r", weekAgo, 2, 1); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertView("o/r", today, 20, 10); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertView("o/r", yesterday, 5, 3); err != nil {
		t.Fatal(err)
	}

	m, err := LoadRepoMetrics(s, "o/r")
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("expected metrics")
	}
	if m.Name != "o/r" {
		t.Errorf("name = %q", m.Name)
	}
	if m.Clones7d < 1 || m.Views7d < 1 {
		t.Errorf("clones7d=%d views7d=%d, want positive rolling windows", m.Clones7d, m.Views7d)
	}
}

func TestLoadRepoMetrics_missingRepo(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "h2h-missing.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	m, err := LoadRepoMetrics(s, "none/here")
	if err != nil {
		t.Fatal(err)
	}
	if m != nil {
		t.Errorf("got %+v, want nil", m)
	}
}

func TestSumDayCountRows_andUtcWindow(t *testing.T) {
	rows := []store.DayRow{
		{Date: "2026-05-01", Count: 3},
		{Date: "2026-05-02", Count: 7},
	}
	if got := sumDayCountRows(rows); got != 10 {
		t.Errorf("sum = %d, want 10", got)
	}
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	from, to := utcWindow(now, 0, 7)
	if to != "2026-05-10" || from != "2026-05-04" {
		t.Errorf("utcWindow(7d) = %q..%q", from, to)
	}
}
