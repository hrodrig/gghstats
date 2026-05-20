package h2h

import "strings"

// Interval selects which traffic window drives H2H score, table, and charts.
type Interval string

const (
	Interval7d    Interval = "7d"
	Interval30d   Interval = "30d"
	IntervalTotal Interval = "total"
)

// ParseInterval normalizes the query param; default is 7d.
func ParseInterval(s string) Interval {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "30d", "30":
		return Interval30d
	case "total", "totals", "all":
		return IntervalTotal
	default:
		return Interval7d
	}
}

// Label is a short UI name for the interval.
func (i Interval) Label() string {
	switch i {
	case Interval30d:
		return "30 days"
	case IntervalTotal:
		return "All time"
	default:
		return "7 days"
	}
}

// HasMomentum reports whether this interval includes a momentum metric and chart.
func (i Interval) HasMomentum() bool {
	return i != IntervalTotal
}

// MomentumWindowDays is the rolling window size for momentum (0 if disabled).
func (i Interval) MomentumWindowDays() int {
	switch i {
	case Interval30d:
		return 30
	case Interval7d:
		return 7
	default:
		return 0
	}
}

// ChartSpanDays is how many days of daily series to show in clone/view charts.
func (i Interval) ChartSpanDays() int {
	switch i {
	case Interval30d:
		return 30
	case IntervalTotal:
		return 0 // caller uses open-ended range
	default:
		return 7
	}
}

// CloneHistoryDays is how far back to load daily clones (includes extra days for momentum rolling windows).
func (i Interval) CloneHistoryDays() int {
	span := i.ChartSpanDays()
	if span == 0 {
		return 0
	}
	if !i.HasMomentum() {
		return span
	}
	return span + 2*i.MomentumWindowDays()
}
