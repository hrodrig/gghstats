package i18n

import (
	"fmt"

	"github.com/hrodrig/gghstats/internal/h2h"
)

// MetricLabelKey maps h2h row keys to translation keys.
func MetricLabelKey(rowKey string) string {
	switch rowKey {
	case "clones_7d":
		return "h2h.metric_clones_7d"
	case "views_7d":
		return "h2h.metric_views_7d"
	case "momentum_7d":
		return "h2h.metric_momentum_7d"
	case "clones_30d":
		return "h2h.metric_clones_30d"
	case "views_30d":
		return "h2h.metric_views_30d"
	case "momentum_30d":
		return "h2h.metric_momentum_30d"
	case "total_clones":
		return "h2h.metric_total_clones"
	case "total_views":
		return "h2h.metric_total_views"
	default:
		return "h2h.metric_" + rowKey
	}
}

// LocalizeResult translates row labels and rebuilds suggestion text for locale.
func (b *Bundle) LocalizeResult(locale string, res h2h.Result, repoA, repoB string, interval h2h.Interval) h2h.Result {
	for i := range res.Rows {
		res.Rows[i].Label = b.T(locale, MetricLabelKey(res.Rows[i].Key))
	}
	if res.Suggest.Show {
		res.Suggest = b.BuildSuggestion(locale, res.DeltaPct, res.LeadsA, repoA, repoB, res.Rows, interval)
	}
	return res
}

// BuildSuggestion mirrors h2h.buildSuggestion with localized rationale.
func (b *Bundle) BuildSuggestion(locale string, delta float64, leadsA bool, repoA, repoB string, rows []h2h.MetricRow, interval h2h.Interval) h2h.Suggestion {
	if delta < 10 {
		return h2h.Suggestion{
			Confidence: "low",
			Rationale:  b.T(locale, "h2h.suggest_close"),
			Show:       true,
		}
	}
	conf := "medium"
	if delta >= 25 {
		conf = "high"
	}
	leader := repoB
	if leadsA {
		leader = repoA
	}
	clonesKey := "clones_7d"
	viewsKey := "views_7d"
	switch interval {
	case h2h.Interval30d:
		clonesKey = "clones_30d"
		viewsKey = "views_30d"
	case h2h.IntervalTotal:
		clonesKey = "total_clones"
		viewsKey = "total_views"
	}
	var rationale string
	if conf == "high" && rowLeadsA(rows, clonesKey) == leadsA {
		metric := b.T(locale, MetricLabelKey(clonesKey))
		rationale = b.Tfmt(locale, "h2h.suggest_clones", map[string]string{"repo": leader, "metric": metric})
	} else if rowLeadsA(rows, viewsKey) == leadsA {
		metric := b.T(locale, MetricLabelKey(viewsKey))
		rationale = b.Tfmt(locale, "h2h.suggest_views", map[string]string{"repo": leader, "metric": metric})
	} else {
		rationale = b.Tfmt(locale, "h2h.suggest_overall", map[string]string{"repo": leader})
	}
	return h2h.Suggestion{Confidence: conf, Rationale: rationale, Show: true}
}

func rowLeadsA(rows []h2h.MetricRow, key string) bool {
	for _, r := range rows {
		if r.Key == key {
			return r.LeadsA
		}
	}
	return false
}

// IntervalLabel returns localized interval name.
func (b *Bundle) IntervalLabel(locale string, iv h2h.Interval) string {
	switch iv {
	case h2h.Interval7d:
		return b.T(locale, "h2h.interval_7d")
	case h2h.Interval30d:
		return b.T(locale, "h2h.interval_30d")
	case h2h.IntervalTotal:
		return b.T(locale, "h2h.interval_total")
	default:
		return iv.Label()
	}
}

// ConfidenceLabel returns localized confidence phrase.
func (b *Bundle) ConfidenceLabel(locale, conf string) string {
	switch conf {
	case "high":
		return b.T(locale, "h2h.confidence_high")
	case "medium":
		return b.T(locale, "h2h.confidence_medium")
	case "low":
		return b.T(locale, "h2h.confidence_low")
	default:
		return conf
	}
}

// H2HError returns a localized H2H error message key result.
func (b *Bundle) H2HError(locale, code string) string {
	switch code {
	case "invalid":
		return b.T(locale, "h2h.err_invalid_repos")
	case "same":
		return b.T(locale, "h2h.err_same_repo")
	case "not_found":
		return b.T(locale, "h2h.err_not_found")
	default:
		return code
	}
}

// ChartSpanLabel localized chart window label.
func (b *Bundle) ChartSpanLabel(locale string, iv h2h.Interval) string {
	return b.IntervalLabel(locale, iv)
}

// MomentumChartLabel localized momentum chart title.
func (b *Bundle) MomentumChartLabel(locale string, iv h2h.Interval) string {
	switch iv {
	case h2h.Interval7d:
		return b.T(locale, "h2h.chart_momentum_7d")
	case h2h.Interval30d:
		return b.T(locale, "h2h.chart_momentum_30d")
	default:
		return ""
	}
}

// LeadsLabel formats "leads: repo".
func (b *Bundle) LeadsLabel(locale, repo string) string {
	return b.Tfmt(locale, "h2h.leads_label", map[string]string{"repo": repo})
}

// LeadPtsLabel formats lead points suffix.
func (b *Bundle) LeadPtsLabel(locale string, pts float64) string {
	return b.Tfmt(locale, "h2h.lead_pts", map[string]string{"pts": fmt.Sprintf("%.0f", pts)})
}
