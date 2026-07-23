package h2h

import (
	"fmt"
	"math"
	"strings"
)

// MetricRow is one row in the comparison table.
type MetricRow struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	WeightPct int    `json:"weight_pct"`
	ValueA    int    `json:"value_a"`
	ValueB    int    `json:"value_b"`
	LeadsA    bool   `json:"leads_a"`
}

// Suggestion is an optional, content-agnostic summary of who leads (subdued in UI).
type Suggestion struct {
	Confidence string `json:"confidence"` // high, medium, low
	Rationale  string `json:"rationale"`
	Show       bool   `json:"show"`
}

// Result is the outcome of comparing two repos.
type Result struct {
	Interval Interval    `json:"interval"`
	RepoA    string      `json:"repo_a"`
	RepoB    string      `json:"repo_b"`
	ScoreA   int         `json:"score_a"`
	ScoreB   int         `json:"score_b"`
	DeltaPct float64     `json:"delta_pct"` // relative gap between scores (0–100)
	LeadsA   bool        `json:"leads_a"`
	Rows     []MetricRow `json:"rows"`
	Suggest  Suggestion  `json:"suggest"`
}

// Compare scores repo A vs B for the selected interval.
func Compare(a, b *RepoMetrics, interval Interval) (Result, bool) {
	if a == nil || b == nil {
		return Result{}, false
	}

	var rows []MetricRow
	var scoreA, scoreB int

	switch interval {
	case Interval30d:
		rows = []MetricRow{
			buildRow("clones_30d", "Clones (30d)", 50, a.Clones30d, b.Clones30d),
			buildRow("views_30d", "Views (30d)", 30, a.Views30d, b.Views30d),
			buildMomentumRow("momentum_30d", "Momentum (30d vs prev 30d)", 20, a.Momentum30d, b.Momentum30d),
		}
		scoreA, scoreB = weightedThreePair(a.Clones30d, a.Views30d, a.Momentum30d, b.Clones30d, b.Views30d, b.Momentum30d, 0.50, 0.30, 0.20)
	case IntervalTotal:
		rows = []MetricRow{
			buildRow("total_clones", "Clones (all time)", 50, a.TotalClones, b.TotalClones),
			buildRow("total_views", "Views (all time)", 50, a.TotalViews, b.TotalViews),
		}
		scoreA, scoreB = weightedTwoPair(a.TotalClones, a.TotalViews, b.TotalClones, b.TotalViews, 0.50, 0.50)
	default:
		interval = Interval7d
		rows = []MetricRow{
			buildRow("clones_7d", "Clones (7d)", 50, a.Clones7d, b.Clones7d),
			buildRow("views_7d", "Views (7d)", 30, a.Views7d, b.Views7d),
			buildMomentumRow("momentum_7d", "Momentum (7d vs prev 7d)", 20, a.Momentum7d, b.Momentum7d),
		}
		scoreA, scoreB = weightedThreePair(a.Clones7d, a.Views7d, a.Momentum7d, b.Clones7d, b.Views7d, b.Momentum7d, 0.50, 0.30, 0.20)
	}

	delta := deltaPct(scoreA, scoreB)
	leadsA := scoreA >= scoreB
	if !leadsA && scoreA == scoreB {
		leadsA = true
	}

	res := Result{
		Interval: interval,
		RepoA:    a.Name,
		RepoB:    b.Name,
		ScoreA:   scoreA,
		ScoreB:   scoreB,
		DeltaPct: delta,
		LeadsA:   leadsA,
		Rows:     rows,
		Suggest:  buildSuggestion(delta, leadsA, a.Name, b.Name, rows, interval),
	}
	return res, true
}

func buildRow(key, label string, weightPct, va, vb int) MetricRow {
	return MetricRow{
		Key:       key,
		Label:     label,
		WeightPct: weightPct,
		ValueA:    va,
		ValueB:    vb,
		LeadsA:    va >= vb,
	}
}

func buildMomentumRow(key, label string, weightPct int, ma, mb float64) MetricRow {
	leadsA := ma >= mb
	return MetricRow{
		Key:       key,
		Label:     label,
		WeightPct: weightPct,
		ValueA:    int(math.Round(ma * 100)),
		ValueB:    int(math.Round(mb * 100)),
		LeadsA:    leadsA,
	}
}

// weightedThreePair returns H2H scores (0–100) as each repo's weighted share of the pair's traffic.
// Count metrics use proportional split; momentum uses non-negative share of growth rates.
func weightedThreePair(c1, v1 int, m1 float64, c2, v2 int, m2 float64, wC, wV, wM float64) (scoreA, scoreB int) {
	sC1, sC2 := sharePair(float64(c1), float64(c2))
	sV1, sV2 := sharePair(float64(v1), float64(v2))
	sM1, sM2 := sharePair(m1, m2)
	rawA := wC*sC1 + wV*sV1 + wM*sM1
	rawB := wC*sC2 + wV*sV2 + wM*sM2
	return int(math.Round(rawA * 100)), int(math.Round(rawB * 100))
}

func weightedTwoPair(c1, v1, c2, v2 int, wC, wV float64) (scoreA, scoreB int) {
	sC1, sC2 := sharePair(float64(c1), float64(c2))
	sV1, sV2 := sharePair(float64(v1), float64(v2))
	rawA := wC*sC1 + wV*sV1
	rawB := wC*sC2 + wV*sV2
	return int(math.Round(rawA * 100)), int(math.Round(rawB * 100))
}

// sharePair splits weight between two non-negative values proportionally (50/50 if both zero).
func sharePair(a, b float64) (shareA, shareB float64) {
	if a < 0 {
		a = 0
	}
	if b < 0 {
		b = 0
	}
	total := a + b
	if total <= 0 {
		return 0.5, 0.5
	}
	return a / total, b / total
}

// deltaPct is the score gap relative to the leader (0–100), not vs the mean.
func deltaPct(scoreA, scoreB int) float64 {
	diff := math.Abs(float64(scoreA - scoreB))
	leader := math.Max(float64(scoreA), float64(scoreB))
	if leader < 1 {
		return 0
	}
	return diff / leader * 100
}

func buildSuggestion(delta float64, leadsA bool, repoA, repoB string, rows []MetricRow, interval Interval) Suggestion {
	if delta < 10 {
		return Suggestion{
			Confidence: "low",
			Rationale:  "Scores are too close to call a winner yet.",
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
	case Interval30d:
		clonesKey = "clones_30d"
		viewsKey = "views_30d"
	case IntervalTotal:
		clonesKey = "total_clones"
		viewsKey = "total_views"
	}

	rationale := fmt.Sprintf("%s leads overall (H2H score) — strong candidate for winner.", leader)
	if conf == "high" && rowLeadsA(rows, clonesKey) == leadsA {
		rationale = fmt.Sprintf("%s leads on %s with a clear gap — strong candidate for winner.", leader, rowLabel(rows, clonesKey))
	} else if rowLeadsA(rows, viewsKey) == leadsA {
		rationale = fmt.Sprintf("%s leads on %s — strong candidate for winner.", leader, rowLabel(rows, viewsKey))
	}

	return Suggestion{
		Confidence: conf,
		Rationale:  rationale,
		Show:       true,
	}
}

func rowLabel(rows []MetricRow, key string) string {
	for _, r := range rows {
		if r.Key == key {
			return r.Label
		}
	}
	return key
}

func rowLeadsA(rows []MetricRow, key string) bool {
	for _, r := range rows {
		if r.Key == key {
			return r.LeadsA
		}
	}
	return false
}

// ParseRepoFullName validates owner/repo.
func ParseRepoFullName(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	owner, repo, ok := strings.Cut(s, "/")
	if !ok || owner == "" || repo == "" || strings.Contains(repo, "/") {
		return "", false
	}
	return owner + "/" + repo, true
}
