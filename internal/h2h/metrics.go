package h2h

import (
	"fmt"
	"math"
	"time"

	"github.com/hrodrig/gghstats/internal/store"
)

// FormatMomentumPct formats a momentum ratio as a signed percent string
// (e.g. +42%, -10%, 0%), matching H2H display rounding.
func FormatMomentumPct(m float64) string {
	pct := int(math.Round(m * 100))
	if pct > 0 {
		return fmt.Sprintf("+%d%%", pct)
	}
	return fmt.Sprintf("%d%%", pct)
}

// RepoMetrics holds comparison inputs for one repository (from SQLite, post-sync).
type RepoMetrics struct {
	Name        string
	Clones7d    int
	Clones30d   int
	TotalClones int
	Views7d     int
	Views30d    int
	TotalViews  int
	Momentum7d  float64 // (last7 - prev7) / max(prev7, 1)
	Momentum30d float64 // (last30 - prev30) / max(prev30, 1)
}

// LoadRepoMetrics loads rolling windows, totals, and momentum from summary + daily rows.
func LoadRepoMetrics(db *store.Store, fullName string) (*RepoMetrics, error) {
	summary, err := db.RepoByName(fullName)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, nil
	}

	now := time.Now().UTC()

	views7d, err := sumClonesOrViews(db.ViewsByRange, fullName, now, 0, 7)
	if err != nil {
		return nil, err
	}
	views30d, err := sumClonesOrViews(db.ViewsByRange, fullName, now, 0, 30)
	if err != nil {
		return nil, err
	}

	mom7, err := cloneMomentumRatio(db, fullName, now, 7)
	if err != nil {
		return nil, err
	}
	mom30, err := cloneMomentumRatio(db, fullName, now, 30)
	if err != nil {
		return nil, err
	}

	return &RepoMetrics{
		Name:        fullName,
		Clones7d:    summary.Clones7d,
		Clones30d:   summary.Clones30d,
		TotalClones: summary.TotalClones,
		Views7d:     views7d,
		Views30d:    views30d,
		TotalViews:  summary.TotalViews,
		Momentum7d:  mom7,
		Momentum30d: mom30,
	}, nil
}

type dayRangeFn func(repo, from, to string) ([]store.DayRow, error)

func sumClonesOrViews(fn dayRangeFn, repo string, now time.Time, daysBackEnd, spanDays int) (int, error) {
	from, to := utcWindow(now, daysBackEnd, spanDays)
	rows, err := fn(repo, from, to)
	if err != nil {
		return 0, err
	}
	return sumDayCountRows(rows), nil
}

func cloneMomentumRatio(db *store.Store, repo string, now time.Time, window int) (float64, error) {
	lastFrom, lastTo := utcWindow(now, 0, window)
	prevFrom, prevTo := utcWindow(now, window, window)

	lastRows, err := db.ClonesByRange(repo, lastFrom, lastTo)
	if err != nil {
		return 0, err
	}
	prevRows, err := db.ClonesByRange(repo, prevFrom, prevTo)
	if err != nil {
		return 0, err
	}
	lastN := sumDayCountRows(lastRows)
	prevN := sumDayCountRows(prevRows)
	denom := float64(prevN)
	if denom < 1 {
		denom = 1
	}
	return float64(lastN-prevN) / denom, nil
}

func utcWindow(now time.Time, daysBackEnd, spanDays int) (from, to string) {
	end := now.AddDate(0, 0, -daysBackEnd)
	start := end.AddDate(0, 0, -(spanDays - 1))
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

func sumDayCountRows(rows []store.DayRow) int {
	sum := 0
	for _, r := range rows {
		sum += r.Count
	}
	return sum
}
