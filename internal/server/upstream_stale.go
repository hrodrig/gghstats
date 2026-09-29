package server

import (
	"time"

	"github.com/hrodrig/gghstats/internal/store"
)

// UpstreamStaleCommunityHelpURL is the banner help link (Community discussions
// search for API/Insights traffic). Do not hard-code a single discussion ID.
const UpstreamStaleCommunityHelpURL = "https://github.com/orgs/community/discussions?discussions_q=API+Insights+traffic"

// ForcedUpstreamStaleSince is the watermark shown when UPSTREAM_STALE_FORCE is on.
const ForcedUpstreamStaleSince = "2026-09-23"

// fleetUpstreamStaleStatus builds the additive upstream_stale JSON/HTML status
// from Config. Banner flag does not gate this status (D-05).
func fleetUpstreamStaleStatus(cfg Config, now time.Time) store.FleetUpstreamStale {
	if cfg.UpstreamStaleForce {
		completed := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
		sinceT, err := time.ParseInLocation("2006-01-02", ForcedUpstreamStaleSince, time.UTC)
		days := 0
		if err == nil {
			days = int(completed.Sub(sinceT).Hours() / 24)
			if days < 1 {
				days = 1
			}
		}
		return store.FleetUpstreamStale{
			Active:        true,
			Since:         ForcedUpstreamStaleSince,
			DaysStuck:     days,
			StuckRepos:    3,
			EligibleRepos: 3,
		}
	}
	if cfg.Store == nil {
		return store.FleetUpstreamStale{}
	}
	lastOK := lastSyncOK(cfg)
	st, err := store.DetectFleetUpstreamStale(cfg.Store, cfg.ReportVisibility, cfg.UpstreamStaleDays, now, lastOK)
	if err != nil {
		return store.FleetUpstreamStale{}
	}
	return st
}

func lastSyncOK(cfg Config) bool {
	if cfg.SyncCoordinator == nil {
		return true
	}
	st := cfg.SyncCoordinator.Status()
	return st.LastFinishedAt != nil && st.LastError == ""
}
