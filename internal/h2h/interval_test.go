package h2h

import (
	"strings"
	"testing"
)

func TestParseInterval(t *testing.T) {
	tests := []struct {
		in   string
		want Interval
	}{
		{"", Interval7d},
		{"7d", Interval7d},
		{"30d", Interval30d},
		{"total", IntervalTotal},
		{"TOTAL", IntervalTotal},
	}
	for _, tt := range tests {
		if got := ParseInterval(tt.in); got != tt.want {
			t.Errorf("ParseInterval(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestInterval_helpers(t *testing.T) {
	if got := Interval7d.Label(); got != "7 days" {
		t.Errorf("7d label = %q", got)
	}
	if got := Interval30d.Label(); got != "30 days" {
		t.Errorf("30d label = %q", got)
	}
	if got := IntervalTotal.Label(); got != "All time" {
		t.Errorf("total label = %q", got)
	}
	if !Interval7d.HasMomentum() || !Interval30d.HasMomentum() {
		t.Error("7d/30d should have momentum")
	}
	if IntervalTotal.HasMomentum() {
		t.Error("total should not have momentum")
	}
	if Interval7d.MomentumWindowDays() != 7 || Interval30d.MomentumWindowDays() != 30 {
		t.Error("momentum window days")
	}
	if IntervalTotal.MomentumWindowDays() != 0 {
		t.Error("total momentum window")
	}
	if Interval7d.ChartSpanDays() != 7 || Interval30d.ChartSpanDays() != 30 {
		t.Error("chart span days")
	}
	if IntervalTotal.ChartSpanDays() != 0 {
		t.Error("total chart span")
	}
	if Interval7d.CloneHistoryDays() != 21 {
		t.Errorf("7d clone history = %d, want 21", Interval7d.CloneHistoryDays())
	}
	if IntervalTotal.CloneHistoryDays() != 0 {
		t.Error("total clone history")
	}
}

func TestCompare_totalNoMomentumRow(t *testing.T) {
	a := &RepoMetrics{Name: "a/r", TotalClones: 1000, TotalViews: 500}
	b := &RepoMetrics{Name: "b/r", TotalClones: 100, TotalViews: 50}
	res, ok := Compare(a, b, IntervalTotal)
	if !ok {
		t.Fatal("expected ok")
	}
	for _, row := range res.Rows {
		if strings.HasPrefix(row.Key, "momentum") {
			t.Errorf("unexpected momentum row %q", row.Key)
		}
	}
}
