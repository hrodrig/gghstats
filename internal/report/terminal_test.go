package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hrodrig/gghstats/internal/store"
)

func TestTerminalOutput(t *testing.T) {
	views := []store.DayRow{
		{Date: "2026-03-20", Count: 10, Uniques: 5},
		{Date: "2026-03-21", Count: 20, Uniques: 8},
	}
	clones := []store.DayRow{
		{Date: "2026-03-20", Count: 50, Uniques: 20},
	}
	refs := []store.ReferrerRow{
		{Date: "2026-03-20", Referrer: "google.com", Count: 40, Uniques: 10},
	}
	paths := []store.PathRow{
		{Date: "2026-03-20", Path: "/hrodrig/pgwd", Title: "pgwd", Count: 100, Uniques: 20},
	}

	var buf bytes.Buffer
	Terminal(&buf, "hrodrig/pgwd", views, clones, refs, paths)
	out := buf.String()

	checks := []string{
		"hrodrig/pgwd",
		"Views",
		"2026-03-20",
		"Total",
		"30", // 10+20 total views
		"13", // 5+8 total uniques
		"Clones",
		"50",
		"google.com",
		"/hrodrig/pgwd",
		"Average",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestTerminalEmpty(t *testing.T) {
	var buf bytes.Buffer
	Terminal(&buf, "owner/repo", nil, nil, nil, nil)
	out := buf.String()

	if !strings.Contains(out, "owner/repo") {
		t.Error("output missing repo name")
	}
	if !strings.Contains(out, "Views") {
		t.Error("output missing Views header")
	}
}
