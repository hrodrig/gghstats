package server

import (
	"math"
	"sort"
	"strconv"

	"github.com/hrodrig/gghstats/internal/store"
)

type cloneStatistics struct {
	Mean              string
	Median            string
	Variance          string
	StandardDeviation string
	Minimum           int
	Maximum           int
	P95               int
}

func calculateCloneStatistics(rows []store.DayRow) *cloneStatistics {
	return calculateCloneStatisticsFor(rows, func(row store.DayRow) int { return row.Count })
}

func calculateUniqueCloneStatistics(rows []store.DayRow) *cloneStatistics {
	return calculateCloneStatisticsFor(rows, func(row store.DayRow) int { return row.Uniques })
}

func calculateCloneStatisticsFor(rows []store.DayRow, value func(store.DayRow) int) *cloneStatistics {
	if len(rows) == 0 {
		return nil
	}

	values := make([]float64, len(rows))
	var sum float64
	for i, row := range rows {
		values[i] = float64(value(row))
		sum += values[i]
	}
	mean := sum / float64(len(values))

	sort.Float64s(values)
	median := values[len(values)/2]
	if len(values)%2 == 0 {
		median = (values[len(values)/2-1] + median) / 2
	}

	var squaredDifferenceSum float64
	for _, value := range values {
		difference := value - mean
		squaredDifferenceSum += difference * difference
	}
	variance := squaredDifferenceSum / float64(len(values))
	p95Index := int(math.Ceil(0.95*float64(len(values)))) - 1

	return &cloneStatistics{
		Mean:              formatCloneStatistic(mean),
		Median:            formatCloneStatistic(median),
		Variance:          formatCloneStatistic(variance),
		StandardDeviation: formatCloneStatistic(math.Sqrt(variance)),
		Minimum:           int(values[0]),
		Maximum:           int(values[len(values)-1]),
		P95:               int(values[p95Index]),
	}
}

func formatCloneStatistic(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}
