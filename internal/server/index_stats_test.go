package server

import (
	"testing"

	"github.com/hrodrig/gghstats/internal/store"
)

func TestCalculateCloneStatistics(t *testing.T) {
	tests := []struct {
		name string
		rows []store.DayRow
		want *cloneStatistics
	}{
		{
			name: "empty",
			want: nil,
		},
		{
			name: "single value",
			rows: []store.DayRow{{Count: 7}},
			want: &cloneStatistics{
				Mean: "7.00", Median: "7.00", Variance: "0.00", StandardDeviation: "0.00",
				Minimum: 7, Maximum: 7, P95: 7,
			},
		},
		{
			name: "zero-filled even window",
			rows: []store.DayRow{{Count: 0}, {Count: 2}, {Count: 4}, {Count: 8}},
			want: &cloneStatistics{
				Mean: "3.50", Median: "3.00", Variance: "8.75", StandardDeviation: "2.96",
				Minimum: 0, Maximum: 8, P95: 8,
			},
		},
		{
			name: "odd window nearest-rank p95",
			rows: []store.DayRow{{Count: 1}, {Count: 3}, {Count: 5}, {Count: 7}, {Count: 100}},
			want: &cloneStatistics{
				Mean: "23.20", Median: "5.00", Variance: "1478.56", StandardDeviation: "38.45",
				Minimum: 1, Maximum: 100, P95: 100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateCloneStatistics(tt.rows); !cloneStatisticsEqual(got, tt.want) {
				t.Fatalf("calculateCloneStatistics() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestCalculateUniqueCloneStatistics(t *testing.T) {
	rows := []store.DayRow{
		{Count: 100, Uniques: 1},
		{Count: 200, Uniques: 3},
		{Count: 300, Uniques: 5},
	}
	want := &cloneStatistics{
		Mean: "3.00", Median: "3.00", Variance: "2.67", StandardDeviation: "1.63",
		Minimum: 1, Maximum: 5, P95: 5,
	}
	if got := calculateUniqueCloneStatistics(rows); !cloneStatisticsEqual(got, want) {
		t.Fatalf("calculateUniqueCloneStatistics() = %#v, want %#v", got, want)
	}
}

func cloneStatisticsEqual(got, want *cloneStatistics) bool {
	if got == nil || want == nil {
		return got == want
	}
	return *got == *want
}
