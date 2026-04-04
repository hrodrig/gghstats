package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hrodrig/gghstats/internal/store"
)

func TestCSVOutput(t *testing.T) {
	views := []store.DayRow{
		{Date: "2026-03-20", Count: 10, Uniques: 5},
		{Date: "2026-03-21", Count: 20, Uniques: 8},
	}
	clones := []store.DayRow{
		{Date: "2026-03-20", Count: 50, Uniques: 20},
	}

	var buf bytes.Buffer
	if err := CSV(&buf, "r", views, clones, nil, nil); err != nil {
		t.Fatalf("CSV() error: %v", err)
	}
	out := buf.String()

	checks := []string{
		"# Views",
		"date,views,unique_visitors",
		"2026-03-20,10,5",
		"2026-03-21,20,8",
		"# Clones",
		"date,clones,unique_cloners",
		"2026-03-20,50,20",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestCSVViewsOnlyNoClones(t *testing.T) {
	views := []store.DayRow{
		{Date: "2026-04-01", Count: 1, Uniques: 1},
	}
	var buf bytes.Buffer
	if err := CSV(&buf, "r", views, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "# Views") || !strings.Contains(out, "# Clones") {
		t.Errorf("expected section headers: %s", out)
	}
}

func TestCSVReferrersOnlyNoViewsClones(t *testing.T) {
	refs := []store.ReferrerRow{
		{Date: "2026-04-01", Referrer: "only-ref", Count: 2, Uniques: 1},
	}
	var buf bytes.Buffer
	if err := CSV(&buf, "r", nil, nil, refs, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "# Referrers") || !strings.Contains(out, "only-ref") {
		t.Fatalf("expected referrers section: %s", out)
	}
}

func TestCSVWithReferrersAndPaths(t *testing.T) {
	refs := []store.ReferrerRow{
		{Date: "2026-03-20", Referrer: "google.com", Count: 40, Uniques: 10},
	}
	paths := []store.PathRow{
		{Date: "2026-03-20", Path: "/repo", Title: "Repo", Count: 100, Uniques: 20},
	}

	var buf bytes.Buffer
	if err := CSV(&buf, "r", nil, nil, refs, paths); err != nil {
		t.Fatalf("CSV() error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "google.com") {
		t.Error("missing referrer")
	}
	if !strings.Contains(out, "/repo") {
		t.Error("missing path")
	}
}
