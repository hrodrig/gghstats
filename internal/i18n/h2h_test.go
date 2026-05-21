package i18n

import (
	"strings"
	"testing"

	"github.com/hrodrig/gghstats/internal/h2h"
)

func TestMetricLabelKey(t *testing.T) {
	if got := MetricLabelKey("clones_7d"); got != "h2h.metric_clones_7d" {
		t.Fatalf("got %q", got)
	}
	if got := MetricLabelKey("custom"); got != "h2h.metric_custom" {
		t.Fatalf("default: got %q", got)
	}
}

func TestBuildSuggestionBranches(t *testing.T) {
	b, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	rows := []h2h.MetricRow{
		{Key: "clones_7d", LeadsA: true},
		{Key: "views_7d", LeadsA: false},
	}

	close := b.BuildSuggestion("en", 5, true, "a/r", "b/r", rows, h2h.Interval7d)
	if close.Confidence != "low" || !close.Show {
		t.Fatalf("close: %+v", close)
	}

	highClones := b.BuildSuggestion("en", 30, true, "a/r", "b/r", rows, h2h.Interval7d)
	if highClones.Confidence != "high" || !strings.Contains(highClones.Rationale, "a/r") {
		t.Fatalf("high clones: %+v", highClones)
	}

	viewsLead := b.BuildSuggestion("en", 15, false, "a/r", "b/r", []h2h.MetricRow{
		{Key: "clones_7d", LeadsA: true},
		{Key: "views_7d", LeadsA: false},
	}, h2h.Interval7d)
	if viewsLead.Confidence != "medium" || !strings.Contains(viewsLead.Rationale, "b/r") {
		t.Fatalf("views: %+v", viewsLead)
	}

	overall := b.BuildSuggestion("en", 20, true, "a/r", "b/r", []h2h.MetricRow{
		{Key: "clones_7d", LeadsA: false},
		{Key: "views_7d", LeadsA: false},
	}, h2h.Interval7d)
	if !strings.Contains(overall.Rationale, "leads overall") && !strings.Contains(overall.Rationale, "a/r") {
		t.Fatalf("overall: %+v", overall)
	}

	b.BuildSuggestion("de", 30, true, "a/r", "b/r", rows, h2h.Interval30d)
	b.BuildSuggestion("es", 30, true, "a/r", "b/r", rows, h2h.IntervalTotal)
}

func TestLocalizeResult(t *testing.T) {
	b, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	res := h2h.Result{
		DeltaPct: 30,
		LeadsA:   true,
		Rows:     []h2h.MetricRow{{Key: "clones_7d", Label: "raw"}},
		Suggest:  h2h.Suggestion{Show: true},
	}
	out := b.LocalizeResult("de", res, "o/a", "o/b", h2h.Interval7d)
	if out.Rows[0].Label != b.T("de", "h2h.metric_clones_7d") {
		t.Fatalf("label: %q", out.Rows[0].Label)
	}
	if out.Suggest.Rationale == "" {
		t.Fatal("expected localized suggestion")
	}
}

func TestIntervalAndConfidenceLabels(t *testing.T) {
	b, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if b.IntervalLabel("de", h2h.Interval7d) != "7 Tage" {
		t.Fatal("7d de")
	}
	if b.IntervalLabel("en", h2h.Interval(``)) == "" {
		// unknown interval uses iv.Label()
	}
	if b.ConfidenceLabel("de", "high") != "hohe Konfidenz" {
		t.Fatal("confidence high de")
	}
	if b.ConfidenceLabel("en", "unknown") != "unknown" {
		t.Fatal("passthrough confidence")
	}
}

func TestH2HErrorAndChartLabels(t *testing.T) {
	b, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if b.H2HError("de", "invalid") != b.T("de", "h2h.err_invalid_repos") {
		t.Fatal("invalid")
	}
	if b.H2HError("en", "xyzzy") != "xyzzy" {
		t.Fatal("unknown code")
	}
	if b.ChartSpanLabel("es", h2h.Interval30d) == "" {
		t.Fatal("chart span")
	}
	if b.MomentumChartLabel("de", h2h.Interval7d) == "" {
		t.Fatal("momentum 7d de")
	}
	if b.MomentumChartLabel("en", h2h.IntervalTotal) != "" {
		t.Fatal("total has no momentum chart label")
	}
	if !strings.Contains(b.LeadsLabel("en", "o/r"), "o/r") {
		t.Fatal("leads label")
	}
	if !strings.Contains(b.LeadPtsLabel("en", 12), "12") {
		t.Fatal("lead pts")
	}
}
