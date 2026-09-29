// Production-rigor cases for fleet upstream_stale (operators with large fleets).
package store

import (
	"fmt"
	"testing"
	"time"
)

func TestDetectFleetUpstreamStale_FailedMetricNotStale(t *testing.T) {
	s := tempDB(t)
	now := time.Date(2026, 9, 28, 15, 0, 0, 0, time.UTC)
	for _, name := range []string{"f/1", "f/2", "f/3"} {
		if err := s.UpsertRepo(name, "", 1, 0, 0, 0, 0, false, false, ""); err != nil {
			t.Fatal(err)
		}
		if err := s.UpsertClone(name, "2026-09-25", 5, 1); err != nil {
			t.Fatal(err)
		}
		if err := s.RecordTrafficMetricFailure(name, "clones", fmt.Errorf("api 502")); err != nil {
			t.Fatal(err)
		}
		if err := s.RecordTrafficMetricFailure(name, "views", fmt.Errorf("api 502")); err != nil {
			t.Fatal(err)
		}
	}
	got, err := DetectFleetUpstreamStale(s, ReportVisibility{}, 3, now, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Active {
		t.Fatalf("failed fetches must not look like upstream freeze: %+v", got)
	}
}

func TestDetectFleetUpstreamStale_ReportScopeExcludesPrivate(t *testing.T) {
	s := tempDB(t)
	now := time.Date(2026, 9, 28, 15, 0, 0, 0, time.UTC)
	for _, name := range []string{"priv/1", "priv/2", "priv/3"} {
		if err := s.UpsertRepoWithVisibility(name, "", 1, 0, 0, 0, 0, false, false, "", VisibilityPrivate); err != nil {
			t.Fatal(err)
		}
		fetched := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
		day := "2026-09-20"
		if err := s.RecordTrafficMetricSuccess(name, "clones", []DayRow{{Date: day, Count: 5, Uniques: 1}}, fetched, day, day); err != nil {
			t.Fatal(err)
		}
		if err := s.RecordTrafficMetricSuccess(name, "views", []DayRow{{Date: day, Count: 5, Uniques: 1}}, fetched, day, day); err != nil {
			t.Fatal(err)
		}
		if err := s.UpsertClone(name, day, 5, 1); err != nil {
			t.Fatal(err)
		}
		if err := s.UpsertView(name, day, 5, 1); err != nil {
			t.Fatal(err)
		}
	}
	got, err := DetectFleetUpstreamStale(s, ReportVisibility{IncludePrivate: false}, 3, now, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Active || got.EligibleRepos != 0 {
		t.Fatalf("private-only fleet must not trigger default report scope: %+v", got)
	}
	got, err = DetectFleetUpstreamStale(s, ReportVisibility{IncludePrivate: true}, 3, now, true)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Active || got.StuckRepos < 3 {
		t.Fatalf("include private should activate: %+v", got)
	}
}

func TestDetectFleetUpstreamStale_LargeFleetFiftyPercent(t *testing.T) {
	s := tempDB(t)
	now := time.Date(2026, 9, 28, 15, 0, 0, 0, time.UTC)
	// 10 eligible: 4 stuck → below 50%; 5 stuck → at 50% and N>=3 → active.
	for i := 1; i <= 10; i++ {
		name := fmt.Sprintf("fleet/%d", i)
		day := "2026-09-27" // fresh
		if i <= 4 {
			day = "2026-09-20" // stale
		}
		seedRepoWithTraffic(t, s, name, day, 3, 3)
	}
	got, err := DetectFleetUpstreamStale(s, ReportVisibility{}, 3, now, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Active {
		t.Fatalf("4/10 must stay inactive: %+v", got)
	}
	seedRepoWithTraffic(t, s, "fleet/5", "2026-09-20", 3, 3)
	got, err = DetectFleetUpstreamStale(s, ReportVisibility{}, 3, now, true)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Active || got.StuckRepos != 5 || got.EligibleRepos != 10 {
		t.Fatalf("5/10 should activate: %+v", got)
	}
	if got.DaysStuck < 1 || got.Since == "" {
		t.Fatalf("want since+days_stuck: %+v", got)
	}
}

func TestDetectFleetUpstreamStale_ClonesOnlyStale(t *testing.T) {
	s := tempDB(t)
	now := time.Date(2026, 9, 28, 15, 0, 0, 0, time.UTC)
	fetched := now
	for _, name := range []string{"c/1", "c/2", "c/3"} {
		if err := s.UpsertRepo(name, "", 1, 0, 0, 0, 0, false, false, ""); err != nil {
			t.Fatal(err)
		}
		if err := s.UpsertClone(name, "2026-09-25", 4, 1); err != nil {
			t.Fatal(err)
		}
		if err := s.UpsertView(name, "2026-09-25", 4, 1); err != nil {
			t.Fatal(err)
		}
		if err := s.RecordTrafficMetricSuccess(name, "clones", []DayRow{{Date: "2026-09-20", Count: 4, Uniques: 1}}, fetched, "2026-09-20", "2026-09-20"); err != nil {
			t.Fatal(err)
		}
		if err := s.RecordTrafficMetricSuccess(name, "views", []DayRow{{Date: "2026-09-27", Count: 4, Uniques: 1}}, fetched, "2026-09-27", "2026-09-27"); err != nil {
			t.Fatal(err)
		}
	}
	got, err := DetectFleetUpstreamStale(s, ReportVisibility{}, 3, now, true)
	if err != nil || !got.Active {
		t.Fatalf("clones-OR-views: want active %+v err=%v", got, err)
	}
}
