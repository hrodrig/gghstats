package h2h

import (
	"sort"
	"time"

	"github.com/hrodrig/gghstats/internal/store"
)

// ChartPoint is one day for Chart.js.
type ChartPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// AlignCloneSeries builds union date labels and parallel count slices for two repos (30d window).
func AlignCloneSeries(a, b []store.DayRow) (labels []string, countsA, countsB []int) {
	return alignDayRows(a, b)
}

// AlignViewSeries builds union date labels and parallel view counts for two repos.
func AlignViewSeries(a, b []store.DayRow) (labels []string, countsA, countsB []int) {
	return alignDayRows(a, b)
}

// AlignMomentumSeries builds union date labels and rolling clone momentum ratios
// ((lastN - prevN) / max(prevN, 1)) for window N (7 or 30). Requires at least 2×window days of history.
func AlignMomentumSeries(a, b []store.DayRow, window int) (labels []string, momA, momB []*float64) {
	if window < 1 {
		return nil, nil, nil
	}
	la, va := cloneMomentumSeries(a, window)
	lb, vb := cloneMomentumSeries(b, window)
	return alignNullableFloatSeries(la, va, lb, vb)
}

// TrimFloatSeriesFrom keeps labels on or after from (YYYY-MM-DD).
func TrimFloatSeriesFrom(labels []string, a, b []*float64, from string) ([]string, []*float64, []*float64) {
	if from == "" || len(labels) == 0 {
		return labels, a, b
	}
	outL := make([]string, 0, len(labels))
	outA := make([]*float64, 0, len(a))
	outB := make([]*float64, 0, len(b))
	for i, d := range labels {
		if d >= from {
			outL = append(outL, d)
			if i < len(a) {
				outA = append(outA, a[i])
			}
			if i < len(b) {
				outB = append(outB, b[i])
			}
		}
	}
	return outL, outA, outB
}

func cloneMomentumSeries(rows []store.DayRow, window int) (labels []string, values []float64) {
	days, counts := denseDailyCounts(rows)
	if len(days) < 2*window {
		return nil, nil
	}
	for i := 2*window - 1; i < len(days); i++ {
		last7 := 0
		for j := i - window + 1; j <= i; j++ {
			last7 += counts[days[j]]
		}
		prev7 := 0
		for j := i - 2*window + 1; j <= i-window; j++ {
			prev7 += counts[days[j]]
		}
		denom := prev7
		if denom < 1 {
			denom = 1
		}
		labels = append(labels, days[i])
		values = append(values, float64(last7-prev7)/float64(denom))
	}
	return labels, values
}

func denseDailyCounts(rows []store.DayRow) (days []string, counts map[string]int) {
	if len(rows) == 0 {
		return nil, nil
	}
	counts = make(map[string]int, len(rows))
	var minD, maxD time.Time
	for i, r := range rows {
		d, err := time.ParseInLocation("2006-01-02", r.Date, time.UTC)
		if err != nil {
			continue
		}
		counts[r.Date] = r.Count
		if i == 0 || d.Before(minD) {
			minD = d
		}
		if i == 0 || d.After(maxD) {
			maxD = d
		}
	}
	for d := minD; !d.After(maxD); d = d.AddDate(0, 0, 1) {
		day := d.Format("2006-01-02")
		days = append(days, day)
		if _, ok := counts[day]; !ok {
			counts[day] = 0
		}
	}
	return days, counts
}

func alignNullableFloatSeries(la []string, va []float64, lb []string, vb []float64) (labels []string, outA, outB []*float64) {
	mA := make(map[string]float64, len(la))
	mB := make(map[string]float64, len(lb))
	set := make(map[string]struct{})
	for i, d := range la {
		mA[d] = va[i]
		set[d] = struct{}{}
	}
	for i, d := range lb {
		mB[d] = vb[i]
		set[d] = struct{}{}
	}
	labels = make([]string, 0, len(set))
	for d := range set {
		labels = append(labels, d)
	}
	sort.Strings(labels)
	outA = make([]*float64, len(labels))
	outB = make([]*float64, len(labels))
	for i, d := range labels {
		if v, ok := mA[d]; ok {
			vv := v
			outA[i] = &vv
		}
		if v, ok := mB[d]; ok {
			vv := v
			outB[i] = &vv
		}
	}
	return labels, outA, outB
}

func alignDayRows(a, b []store.DayRow) (labels []string, outA, outB []int) {
	mA := make(map[string]int, len(a))
	mB := make(map[string]int, len(b))
	set := make(map[string]struct{})
	for _, r := range a {
		mA[r.Date] = r.Count
		set[r.Date] = struct{}{}
	}
	for _, r := range b {
		mB[r.Date] = r.Count
		set[r.Date] = struct{}{}
	}
	labels = make([]string, 0, len(set))
	for d := range set {
		labels = append(labels, d)
	}
	sort.Strings(labels)
	outA = make([]int, len(labels))
	outB = make([]int, len(labels))
	for i, d := range labels {
		outA[i] = mA[d]
		outB[i] = mB[d]
	}
	return labels, outA, outB
}
