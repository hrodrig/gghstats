package h2h

import (
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/store"
)

func TestCloneMomentumSeries_increasing(t *testing.T) {
	var rows []store.DayRow
	start := mustParseDay(t, "2026-04-01")
	for i := 0; i < 14; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		n := 5
		if i >= 7 {
			n = 15
		}
		rows = append(rows, store.DayRow{Date: d, Count: n})
	}
	labels, vals := cloneMomentumSeries(rows, 7)
	if len(labels) == 0 || len(vals) == 0 {
		t.Fatal("expected momentum points")
	}
	last := vals[len(vals)-1]
	if last <= 0 {
		t.Errorf("expected positive momentum, got %v", last)
	}
}

func TestTrimFloatSeriesFrom(t *testing.T) {
	labels := []string{"2026-05-10", "2026-05-11", "2026-05-12"}
	v1, v2, v3 := 1.0, 2.0, 3.0
	a := []*float64{&v1, &v2, &v3}
	b := []*float64{nil, &v2, nil}
	outL, outA, outB := TrimFloatSeriesFrom(labels, a, b, "2026-05-11")
	if len(outL) != 2 || outL[0] != "2026-05-11" {
		t.Fatalf("labels = %v", outL)
	}
	if outA[0] == nil || *outA[0] != 2.0 {
		t.Fatalf("outA = %v", outA)
	}
	if outB[0] == nil || *outB[0] != 2.0 {
		t.Fatalf("outB = %v", outB)
	}
}

func TestAlignCloneAndViewSeries(t *testing.T) {
	a := []store.DayRow{
		{Date: "2026-05-10", Count: 5},
		{Date: "2026-05-12", Count: 8},
	}
	b := []store.DayRow{
		{Date: "2026-05-11", Count: 3},
		{Date: "2026-05-12", Count: 2},
	}
	labels, ca, cb := AlignCloneSeries(a, b)
	if len(labels) != 3 || labels[0] != "2026-05-10" {
		t.Fatalf("clone labels = %v", labels)
	}
	if ca[0] != 5 || ca[1] != 0 || cb[2] != 2 {
		t.Fatalf("clone counts A=%v B=%v", ca, cb)
	}
	vLabels, va, vb := AlignViewSeries(a, b)
	if len(vLabels) != len(labels) || va[2] != 8 {
		t.Fatalf("view align = %v %v %v", vLabels, va, vb)
	}
}

func TestAlignMomentumSeries_unionLabels(t *testing.T) {
	a := []store.DayRow{{Date: "2026-04-01", Count: 10}}
	b := []store.DayRow{{Date: "2026-04-02", Count: 10}}
	labels, _, _ := AlignMomentumSeries(a, b, 7)
	if len(labels) != 0 {
		t.Errorf("expected no labels with insufficient history, got %v", labels)
	}
}

func mustParseDay(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.ParseInLocation("2006-01-02", s, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
