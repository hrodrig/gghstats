package demo

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hrodrig/gghstats/internal/store"
)

// ApplyUpstreamStaleFreeze marks every non-hidden repo's clones/views metric
// state as successful but stalled (latest_observed older than K=3), while
// leaving day tables intact so fleet eligibility still sees non-zero traffic.
//
// Dogfood only: makes DetectFleetUpstreamStale fire without calling GitHub.
// Call after SeedIfEmpty when GGHSTATS_DEMO_UPSTREAM_STALE is enabled.
func ApplyUpstreamStaleFreeze(db *store.Store) error {
	if db == nil {
		return fmt.Errorf("demo upstream_stale: nil store")
	}
	now := time.Now().UTC()
	completed := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	// Stale vs default K=3: observed <= completed-3.
	observed := completed.AddDate(0, 0, -4).Format("2006-01-02")
	repos, err := db.ListRepos("name", "asc")
	if err != nil {
		return fmt.Errorf("demo upstream_stale list: %w", err)
	}
	if len(repos) == 0 {
		return fmt.Errorf("demo upstream_stale: no repos (seed demo data first)")
	}
	for _, r := range repos {
		for _, metric := range []string{"clones", "views"} {
			if err := db.UpsertTrafficMetricStateSuccess(r.Name, metric, observed, now); err != nil {
				return fmt.Errorf("demo upstream_stale %s %s: %w", r.Name, metric, err)
			}
		}
	}
	slog.Info("demo mode: applied upstream_stale freeze watermark",
		"since", observed, "repos", len(repos))
	return nil
}
