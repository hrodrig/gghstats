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

func TestSanitizeCSVCell(t *testing.T) {
	tests := []struct {
		name string
		cell string
		want string
	}{
		{name: "plain host is untouched", cell: "github.com", want: "github.com"},
		{name: "path is untouched", cell: "/hrodrig/pgwd", want: "/hrodrig/pgwd"},
		{name: "empty is untouched", cell: "", want: ""},
		{name: "equals prefix", cell: "=cmd|'/C calc'!A0", want: "'=cmd|'/C calc'!A0"},
		{name: "plus prefix", cell: "+1+1", want: "'+1+1"},
		{name: "minus prefix", cell: "-1+1", want: "'-1+1"},
		{name: "at prefix", cell: "@SUM(1+1)", want: "'@SUM(1+1)"},
		{name: "tab prefix", cell: "\t=1+1", want: "'\t=1+1"},
		{name: "carriage return prefix", cell: "\r=1+1", want: "'\r=1+1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeCSVCell(tt.cell); got != tt.want {
				t.Errorf("sanitizeCSVCell(%q) = %q, want %q", tt.cell, got, tt.want)
			}
		})
	}
}

func TestCSVNeutralizesSpreadsheetFormulas(t *testing.T) {
	refs := []store.ReferrerRow{
		{Date: "2026-03-20", Referrer: "=1+1", Count: 40, Uniques: 10},
	}
	paths := []store.PathRow{
		{Date: "2026-03-20", Path: "/repo", Title: "@SUM(1+1)", Count: 100, Uniques: 20},
	}

	var buf bytes.Buffer
	if err := CSV(&buf, "r", nil, nil, refs, paths); err != nil {
		t.Fatalf("CSV() error: %v", err)
	}
	out := buf.String()

	escaped := []string{"'=1+1", "'@SUM(1+1)"}
	for _, want := range escaped {
		if !strings.Contains(out, want) {
			t.Errorf("exported CSV is missing the neutralized cell %q: %s", want, out)
		}
	}

	raw := []string{",=1+1", ",@SUM(1+1)"}
	for _, unwanted := range raw {
		if strings.Contains(out, unwanted) {
			t.Errorf("exported CSV contains an unescaped formula %q: %s", unwanted, out)
		}
	}
}
