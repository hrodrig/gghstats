package store

import (
	"fmt"
	"time"
)

// FleetUpstreamStale is the fleet-level GitHub traffic freeze signal.
// Distinct from per-repo per-metric freshness (delayed/missing/failed).
type FleetUpstreamStale struct {
	Active        bool   `json:"active"`
	Since         string `json:"since,omitempty"`
	DaysStuck     int    `json:"days_stuck,omitempty"`
	StuckRepos    int    `json:"stuck_repos"`
	EligibleRepos int    `json:"eligible_repos"`
}

const upstreamStaleEligibleWindowDays = 14

// DetectFleetUpstreamStale classifies report-scoped repos and applies fleet
// thresholds (K days, N>=3, stuck*2>=eligible). K=0 disables detection.
// lastSyncOK must be true (successful traffic sync); otherwise Active is false.
func DetectFleetUpstreamStale(db *Store, scope ReportVisibility, k int, now time.Time, lastSyncOK bool) (FleetUpstreamStale, error) {
	out := FleetUpstreamStale{}
	if db == nil {
		return out, fmt.Errorf("nil store")
	}
	if k <= 0 || !lastSyncOK {
		return out, nil
	}

	nowUTC := now.UTC()
	completed := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	staleCutoff := completed.AddDate(0, 0, -k).Format("2006-01-02")
	windowEnd := completed.Format("2006-01-02")
	windowStart := completed.AddDate(0, 0, -(upstreamStaleEligibleWindowDays - 1)).Format("2006-01-02")

	repos, err := db.ListReportRepos(scope, "name", "asc")
	if err != nil {
		return out, err
	}

	var (
		stuckCount    int
		eligibleCount int
		sinceMax      string
	)
	for _, repo := range repos {
		eligible, err := repoHasNonZeroTrafficInWindow(db, repo.Name, windowStart, windowEnd)
		if err != nil {
			return out, err
		}
		if !eligible {
			continue
		}
		eligibleCount++

		clonesStale, clonesObs, err := metricIsUpstreamStale(db, repo.Name, "clones", staleCutoff)
		if err != nil {
			return out, err
		}
		viewsStale, viewsObs, err := metricIsUpstreamStale(db, repo.Name, "views", staleCutoff)
		if err != nil {
			return out, err
		}
		if !clonesStale && !viewsStale {
			continue
		}
		stuckCount++
		if clonesStale && clonesObs != "" && clonesObs > sinceMax {
			sinceMax = clonesObs
		}
		if viewsStale && viewsObs != "" && viewsObs > sinceMax {
			sinceMax = viewsObs
		}
	}

	out.StuckRepos = stuckCount
	out.EligibleRepos = eligibleCount
	if stuckCount < 3 || stuckCount*2 < eligibleCount {
		return out, nil
	}

	out.Active = true
	out.Since = sinceMax
	if sinceMax != "" {
		sinceDay, err := time.ParseInLocation("2006-01-02", sinceMax, time.UTC)
		if err == nil {
			// Completed UTC days from watermark through latest completed day.
			days := int(completed.Sub(sinceDay).Hours() / 24)
			if days < 0 {
				days = 0
			}
			out.DaysStuck = days
		}
	}
	return out, nil
}

func metricIsUpstreamStale(db *Store, repo, metric, staleCutoff string) (stale bool, latestObserved string, err error) {
	st, err := db.TrafficMetricState(repo, metric)
	if err != nil {
		return false, "", err
	}
	if st.LastStatus != "success" || st.LatestObservedDate == "" {
		return false, "", nil
	}
	if st.LatestObservedDate <= staleCutoff {
		return true, st.LatestObservedDate, nil
	}
	return false, st.LatestObservedDate, nil
}

func repoHasNonZeroTrafficInWindow(db *Store, repo, from, to string) (bool, error) {
	var clones, views int
	err := db.db.QueryRow(
		`SELECT COALESCE(SUM(count), 0) FROM clones WHERE repo = ? AND date >= ? AND date <= ?`,
		repo, from, to,
	).Scan(&clones)
	if err != nil {
		return false, err
	}
	if clones > 0 {
		return true, nil
	}
	err = db.db.QueryRow(
		`SELECT COALESCE(SUM(count), 0) FROM views WHERE repo = ? AND date >= ? AND date <= ?`,
		repo, from, to,
	).Scan(&views)
	if err != nil {
		return false, err
	}
	return views > 0, nil
}
