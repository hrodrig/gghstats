package server

import (
	"time"

	"github.com/hrodrig/gghstats/internal/store"
)

// UpstreamStaleCommunityHelpURL is the banner help link (Community discussions
// search for API/Insights traffic). Do not hard-code a single discussion ID.
const UpstreamStaleCommunityHelpURL = "https://github.com/orgs/community/discussions?discussions_q=API+Insights+traffic"

// fleetUpstreamStaleStatus builds the additive upstream_stale JSON/HTML status
// from Config. Banner flag does not gate this status (D-05).
func fleetUpstreamStaleStatus(cfg Config, now time.Time) store.FleetUpstreamStale {
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
