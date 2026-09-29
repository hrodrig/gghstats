package store

import (
	"strconv"
	"testing"
	"time"
)

func seedRepoWithTraffic(t *testing.T, s *Store, name, observedDay string, clones, views int) {
	t.Helper()
	if err := s.UpsertRepo(name, "", 1, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	fetched := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	if err := s.RecordTrafficMetricSuccess(name, "clones", []DayRow{{Date: observedDay, Count: clones, Uniques: 1}}, fetched, observedDay, observedDay); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordTrafficMetricSuccess(name, "views", []DayRow{{Date: observedDay, Count: views, Uniques: 1}}, fetched, observedDay, observedDay); err != nil {
		t.Fatal(err)
	}
	// Ensure 14d eligible window has non-zero counts on the observed day.
	if err := s.UpsertClone(name, observedDay, clones, 1); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertView(name, observedDay, views, 1); err != nil {
		t.Fatal(err)
	}
}

func TestDetectFleetUpstreamStale_KZeroDisables(t *testing.T) {
	s := tempDB(t)
	now := time.Date(2026, 9, 28, 15, 0, 0, 0, time.UTC)
	for _, name := range []string{"a/1", "a/2", "a/3"} {
		seedRepoWithTraffic(t, s, name, "2026-09-20", 5, 5)
	}
	got, err := DetectFleetUpstreamStale(s, ReportVisibility{}, 0, now, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Active {
		t.Fatalf("K=0 should disable: %+v", got)
	}
}

func TestDetectFleetUpstreamStale_LastSyncNotOK(t *testing.T) {
	s := tempDB(t)
	now := time.Date(2026, 9, 28, 15, 0, 0, 0, time.UTC)
	for _, name := range []string{"a/1", "a/2", "a/3"} {
		seedRepoWithTraffic(t, s, name, "2026-09-20", 5, 5)
	}
	got, err := DetectFleetUpstreamStale(s, ReportVisibility{}, 3, now, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Active {
		t.Fatalf("lastSyncOK=false should not activate: %+v", got)
	}
}

func TestDetectFleetUpstreamStale_HappyPathActive(t *testing.T) {
	s := tempDB(t)
	now := time.Date(2026, 9, 28, 15, 0, 0, 0, time.UTC)
	// completed=2026-09-27, K=3 → cutoff=2026-09-24; observed 2026-09-23 is stale.
	for _, name := range []string{"o/r1", "o/r2", "o/r3"} {
		seedRepoWithTraffic(t, s, name, "2026-09-23", 10, 10)
	}
	got, err := DetectFleetUpstreamStale(s, ReportVisibility{}, 3, now, true)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Active || got.Since != "2026-09-23" || got.StuckRepos != 3 || got.EligibleRepos != 3 {
		t.Fatalf("got %+v", got)
	}
	// completed (2026-09-27) - since (2026-09-23) = 4 completed UTC days
	if got.DaysStuck != 4 {
		t.Fatalf("DaysStuck = %d, want 4", got.DaysStuck)
	}
}

func TestDetectFleetUpstreamStale_EmptyNeverNotStale(t *testing.T) {
	s := tempDB(t)
	now := time.Date(2026, 9, 28, 15, 0, 0, 0, time.UTC)
	if err := s.UpsertRepo("quiet/r", "", 0, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	// Never fetched — not eligible (no day rows) and not stuck.
	got, err := DetectFleetUpstreamStale(s, ReportVisibility{}, 3, now, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Active || got.StuckRepos != 0 || got.EligibleRepos != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestDetectFleetUpstreamStale_Thresholds(t *testing.T) {
	now := time.Date(2026, 9, 28, 15, 0, 0, 0, time.UTC)
	cases := []struct {
		name       string
		stuckN     int
		freshN     int
		wantActive bool
	}{
		{"2stuck_3elig", 2, 1, false},
		{"3stuck_6elig", 3, 3, true},
		{"3stuck_7elig", 3, 4, false},
		{"4stuck_7elig", 4, 3, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tempDB(t)
			i := 0
			for n := 0; n < tc.stuckN; n++ {
				i++
				seedRepoWithTraffic(t, s, "s/"+strconv.Itoa(i), "2026-09-20", 5, 5)
			}
			for n := 0; n < tc.freshN; n++ {
				i++
				seedRepoWithTraffic(t, s, "f/"+strconv.Itoa(i), "2026-09-27", 5, 5)
			}
			got, err := DetectFleetUpstreamStale(s, ReportVisibility{}, 3, now, true)
			if err != nil {
				t.Fatal(err)
			}
			if got.Active != tc.wantActive {
				t.Fatalf("Active=%v want %v; stuck=%d elig=%d", got.Active, tc.wantActive, got.StuckRepos, got.EligibleRepos)
			}
			wantElig := tc.stuckN + tc.freshN
			if got.EligibleRepos != wantElig || got.StuckRepos != tc.stuckN {
				t.Fatalf("counts stuck=%d elig=%d want stuck=%d elig=%d", got.StuckRepos, got.EligibleRepos, tc.stuckN, wantElig)
			}
		})
	}
}

func TestDetectFleetUpstreamStale_ORMetricsViewsFreshClonesStale(t *testing.T) {
	s := tempDB(t)
	now := time.Date(2026, 9, 28, 15, 0, 0, 0, time.UTC)
	fetched := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	for i, name := range []string{"o/a", "o/b", "o/c"} {
		_ = i
		if err := s.UpsertRepo(name, "", 1, 0, 0, 0, 0, false, false, ""); err != nil {
			t.Fatal(err)
		}
		// Clones stuck at 2026-09-20; views fresh at 2026-09-27.
		if err := s.RecordTrafficMetricSuccess(name, "clones", []DayRow{{Date: "2026-09-20", Count: 8, Uniques: 2}}, fetched, "2026-09-20", "2026-09-20"); err != nil {
			t.Fatal(err)
		}
		if err := s.RecordTrafficMetricSuccess(name, "views", []DayRow{{Date: "2026-09-27", Count: 3, Uniques: 1}}, fetched, "2026-09-27", "2026-09-27"); err != nil {
			t.Fatal(err)
		}
		if err := s.UpsertClone(name, "2026-09-20", 8, 2); err != nil {
			t.Fatal(err)
		}
		if err := s.UpsertView(name, "2026-09-27", 3, 1); err != nil {
			t.Fatal(err)
		}
	}
	got, err := DetectFleetUpstreamStale(s, ReportVisibility{}, 3, now, true)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Active || got.StuckRepos != 3 {
		t.Fatalf("OR metric stuck want active: %+v", got)
	}
	if got.Since != "2026-09-20" {
		t.Fatalf("Since = %q, want max among stuck metrics 2026-09-20", got.Since)
	}
}

func TestDetectFleetUpstreamStale_QuietRepoExcluded(t *testing.T) {
	s := tempDB(t)
	now := time.Date(2026, 9, 28, 15, 0, 0, 0, time.UTC)
	for _, name := range []string{"o/1", "o/2", "o/3"} {
		seedRepoWithTraffic(t, s, name, "2026-09-20", 5, 5)
	}
	if err := s.UpsertRepo("quiet/zero", "", 0, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	// Explicit zero rows in window — still not eligible (sum == 0).
	if err := s.UpsertClone("quiet/zero", "2026-09-25", 0, 0); err != nil {
		t.Fatal(err)
	}
	got, err := DetectFleetUpstreamStale(s, ReportVisibility{}, 3, now, true)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Active || got.EligibleRepos != 3 {
		t.Fatalf("quiet excluded: %+v", got)
	}
}

func TestDetectFleetUpstreamStale_AutoClearWhenObservedAdvances(t *testing.T) {
	s := tempDB(t)
	now := time.Date(2026, 9, 28, 15, 0, 0, 0, time.UTC)
	for _, name := range []string{"o/1", "o/2", "o/3"} {
		seedRepoWithTraffic(t, s, name, "2026-09-20", 5, 5)
	}
	got, err := DetectFleetUpstreamStale(s, ReportVisibility{}, 3, now, true)
	if err != nil || !got.Active {
		t.Fatalf("precondition stuck: %+v err=%v", got, err)
	}
	fetched := time.Date(2026, 9, 28, 18, 0, 0, 0, time.UTC)
	for _, name := range []string{"o/1", "o/2", "o/3"} {
		if err := s.RecordTrafficMetricSuccess(name, "clones", []DayRow{{Date: "2026-09-27", Count: 2, Uniques: 1}}, fetched, "2026-09-27", "2026-09-27"); err != nil {
			t.Fatal(err)
		}
		if err := s.RecordTrafficMetricSuccess(name, "views", []DayRow{{Date: "2026-09-27", Count: 2, Uniques: 1}}, fetched, "2026-09-27", "2026-09-27"); err != nil {
			t.Fatal(err)
		}
	}
	got, err = DetectFleetUpstreamStale(s, ReportVisibility{}, 3, now, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Active {
		t.Fatalf("should auto-clear when latest_observed advances: %+v", got)
	}
}

func TestDetectFleetUpstreamStale_NilStore(t *testing.T) {
	_, err := DetectFleetUpstreamStale(nil, ReportVisibility{}, 3, time.Now().UTC(), true)
	if err == nil {
		t.Fatal("want error for nil store")
	}
}

func TestDaysStuckFromSince(t *testing.T) {
	completed := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)
	if daysStuckFromSince(completed, "") != 0 {
		t.Fatal("empty since")
	}
	if daysStuckFromSince(completed, "not-a-date") != 0 {
		t.Fatal("invalid since")
	}
	if got := daysStuckFromSince(completed, "2026-09-23"); got != 4 {
		t.Fatalf("days=%d want 4", got)
	}
	if daysStuckFromSince(completed, "2026-09-30") != 0 {
		t.Fatal("future since should clamp to 0")
	}
}

func TestRepoUpstreamStaleObs_ViewsOnly(t *testing.T) {
	s := tempDB(t)
	now := time.Date(2026, 9, 28, 15, 0, 0, 0, time.UTC)
	completed := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)
	staleCutoff := completed.AddDate(0, 0, -3).Format("2006-01-02")
	name := "o/views"
	if err := s.UpsertRepo(name, "", 1, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	fetched := now
	// Fresh clones, stale views.
	if err := s.RecordTrafficMetricSuccess(name, "clones", []DayRow{{Date: "2026-09-27", Count: 3, Uniques: 1}}, fetched, "2026-09-27", "2026-09-27"); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordTrafficMetricSuccess(name, "views", []DayRow{{Date: "2026-09-20", Count: 3, Uniques: 1}}, fetched, "2026-09-20", "2026-09-20"); err != nil {
		t.Fatal(err)
	}
	stuck, obs, err := repoUpstreamStaleObs(s, name, staleCutoff)
	if err != nil || !stuck || obs != "2026-09-20" {
		t.Fatalf("stuck=%v obs=%q err=%v", stuck, obs, err)
	}
}

func TestMetricIsUpstreamStale_NeverAndFresh(t *testing.T) {
	s := tempDB(t)
	name := "o/m"
	if err := s.UpsertRepo(name, "", 1, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	stale, obs, err := metricIsUpstreamStale(s, name, "clones", "2026-09-20")
	if err != nil || stale || obs != "" {
		t.Fatalf("never: stale=%v obs=%q err=%v", stale, obs, err)
	}
	fetched := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	if err := s.RecordTrafficMetricSuccess(name, "clones", []DayRow{{Date: "2026-09-27", Count: 1, Uniques: 1}}, fetched, "2026-09-27", "2026-09-27"); err != nil {
		t.Fatal(err)
	}
	stale, obs, err = metricIsUpstreamStale(s, name, "clones", "2026-09-20")
	if err != nil || stale || obs != "2026-09-27" {
		t.Fatalf("fresh: stale=%v obs=%q err=%v", stale, obs, err)
	}
}

func TestRepoHasNonZeroTrafficViewsOnly(t *testing.T) {
	s := tempDB(t)
	name := "o/v"
	if err := s.UpsertRepo(name, "", 1, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertView(name, "2026-09-25", 4, 2); err != nil {
		t.Fatal(err)
	}
	ok, err := repoHasNonZeroTrafficInWindow(s, name, "2026-09-20", "2026-09-27")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestUpsertTrafficMetricStateSuccess(t *testing.T) {
	s := tempDB(t)
	if err := s.UpsertRepo("o/r", "", 1, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertClone("o/r", "2026-09-27", 5, 2); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := s.UpsertTrafficMetricStateSuccess("o/r", "clones", "2026-09-20", now); err != nil {
		t.Fatal(err)
	}
	st, err := s.TrafficMetricState("o/r", "clones")
	if err != nil {
		t.Fatal(err)
	}
	if st.LastStatus != "success" || st.LatestObservedDate != "2026-09-20" {
		t.Fatalf("%+v", st)
	}
	if err := s.UpsertTrafficMetricStateSuccess("o/r", "badmetric", "2026-09-20", now); err == nil {
		t.Fatal("want error for bad metric")
	}
}
