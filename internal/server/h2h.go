package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hrodrig/gghstats/internal/h2h"
	"github.com/hrodrig/gghstats/internal/i18n"
	"github.com/hrodrig/gghstats/internal/store"
	"github.com/hrodrig/gghstats/internal/version"
)

type h2hRepoOption struct {
	Name string
}

type h2hPageData struct {
	localeBinder
	Repos              []h2hRepoOption
	RepoA              string
	RepoB              string
	Interval           string
	IntervalLabel      string
	ShowMomentumChart  bool
	ChartSpanLabel     string
	ChartClonesLabel   string
	ChartViewsLabel    string
	MomentumChartLabel string
	MomentumFootnote   string
	Error              string
	Compared           bool
	Result             h2h.Result
	ScoreALeadLine     string
	ScoreBLeadLine     string
	ConfidenceLabel    string
	ChartJSON          template.JS
	HasCharts          bool
	FormAction         string
}

func handleH2HPage(cfg Config, db *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/h2h" {
			lb := bindPageLocale(r, cfg)
			writeBrutalistNotFound(w, r, tmpl, cfg, lb.T("not_found.title"), lb.T("not_found.heading"), r.URL.Path, lb.T("not_found.detail"))
			return
		}

		lb := bindPageLocale(r, cfg)
		bundle := i18n.MustLoad()
		loc := lb.Locale

		repos, err := db.ListRepos("name", "asc")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rawA := r.URL.Query().Get("a")
		rawB := r.URL.Query().Get("b")
		interval := h2h.ParseInterval(r.URL.Query().Get("w"))
		data := newH2HPageData(lb, repos, rawA, rawB, interval)

		if err := applyH2HComparison(&data, bundle, loc, db, rawA, rawB, interval); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		content := executeTemplate(tmpl, "h2h", data)
		renderLayout(w, r, tmpl, cfg, layoutData{
			Title:   lb.T("h2h.title"),
			PageID:  "h2h",
			Version: version.Version,
			Breadcrumbs: []breadcrumb{
				{Label: lb.T("nav.home"), URL: "/"},
				{Label: lb.T("nav.h2h"), URL: ""},
			},
			Content: content,
		})
	}
}

func newH2HPageData(lb localeBinder, repos []store.RepoSummary, rawA, rawB string, interval h2h.Interval) h2hPageData {
	opts := make([]h2hRepoOption, len(repos))
	for i, rp := range repos {
		opts[i] = h2hRepoOption{Name: rp.Name}
	}
	span := lb.T("h2h.interval_7d")
	switch interval {
	case h2h.Interval30d:
		span = lb.T("h2h.interval_30d")
	case h2h.IntervalTotal:
		span = lb.T("h2h.interval_total")
	}
	return h2hPageData{
		localeBinder:       lb,
		Repos:              opts,
		RepoA:              strings.TrimSpace(rawA),
		RepoB:              strings.TrimSpace(rawB),
		Interval:           string(interval),
		IntervalLabel:      lb.T("h2h.interval_" + string(interval)),
		ShowMomentumChart:  interval.HasMomentum(),
		ChartSpanLabel:     span,
		ChartClonesLabel:   lb.Tfmt("h2h.chart_clones", map[string]string{"span": span}),
		ChartViewsLabel:    lb.Tfmt("h2h.chart_views", map[string]string{"span": span}),
		MomentumChartLabel: i18n.MustLoad().MomentumChartLabel(lb.Locale, interval),
		MomentumFootnote:   lb.T("h2h.chart_momentum_footnote"),
		FormAction:         "/h2h",
	}
}

// applyH2HComparison fills comparison fields when query params request a pair. Returns an error for load failures.
func applyH2HComparison(data *h2hPageData, bundle *i18n.Bundle, locale string, db *store.Store, rawA, rawB string, interval h2h.Interval) error {
	if rawA == "" && rawB == "" {
		return nil
	}
	repoA, okA := h2h.ParseRepoFullName(rawA)
	repoB, okB := h2h.ParseRepoFullName(rawB)
	if !okA || !okB {
		data.Error = bundle.H2HError(locale, "invalid")
		return nil
	}
	if repoA == repoB {
		data.Error = bundle.H2HError(locale, "same")
		return nil
	}
	return populateH2HComparison(data, bundle, locale, db, repoA, repoB, interval)
}

func populateH2HComparison(data *h2hPageData, bundle *i18n.Bundle, locale string, db *store.Store, repoA, repoB string, interval h2h.Interval) error {
	mA, err := h2h.LoadRepoMetrics(db, repoA)
	if err != nil {
		return err
	}
	mB, err := h2h.LoadRepoMetrics(db, repoB)
	if err != nil {
		return err
	}
	if mA == nil || mB == nil {
		data.Error = bundle.H2HError(locale, "not_found")
		return nil
	}
	res, ok := h2h.Compare(mA, mB, interval)
	if !ok {
		return nil
	}
	res = bundle.LocalizeResult(locale, res, repoA, repoB, interval)
	data.Compared = true
	data.Result = res
	data.RepoA = repoA
	data.RepoB = repoB
	data.IntervalLabel = bundle.IntervalLabel(locale, interval)
	shareFull := bundle.Tfmt(locale, "h2h.h2h_share", map[string]string{"interval": data.IntervalLabel})
	data.ScoreALeadLine = h2hScoreSubline(bundle, locale, shareFull, res, true)
	data.ScoreBLeadLine = h2hScoreSubline(bundle, locale, shareFull, res, false)
	data.ConfidenceLabel = bundle.ConfidenceLabel(locale, res.Suggest.Confidence)
	if chartJS, ok := buildH2HChartJSON(db, repoA, repoB, interval); ok {
		data.ChartJSON = chartJS
		data.HasCharts = true
	}
	return nil
}

// h2hScoreSubline is the muted line under a score card: share label, plus margin only on the leader when the gap is clear.
func h2hScoreSubline(bundle *i18n.Bundle, locale, shareFull string, res h2h.Result, forRepoA bool) string {
	if res.ScoreA == res.ScoreB || res.DeltaPct < 10 {
		return shareFull
	}
	isLeader := res.LeadsA == forRepoA
	if !isLeader {
		return shareFull
	}
	return shareFull + " · " + bundle.LeadMarginLabel(locale, res.DeltaPct)
}

func buildH2HChartJSON(db *store.Store, repoA, repoB string, interval h2h.Interval) (template.JS, bool) {
	now := time.Now().UTC()
	to := now.Format("2006-01-02")
	from, cloneFetchFrom := h2hChartFromDates(now, interval)

	clA, err1 := db.ClonesByRange(repoA, cloneFetchFrom, to)
	clB, err2 := db.ClonesByRange(repoB, cloneFetchFrom, to)
	vA, err3 := db.ViewsByRange(repoA, from, to)
	vB, err4 := db.ViewsByRange(repoB, from, to)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return template.JS("{}"), false
	}

	clAChart, clBChart := trimDayRowsFrom(clA, from), trimDayRowsFrom(clB, from)
	vAChart, vBChart := vA, vB

	clLabels, clAcounts, clBcounts := h2h.AlignCloneSeries(clAChart, clBChart)
	vLabels, vAcounts, vBcounts := h2h.AlignViewSeries(vAChart, vBChart)

	var mLabels []string
	var mA, mB []*float64
	if interval.HasMomentum() {
		mLabels, mA, mB = h2h.AlignMomentumSeries(clA, clB, interval.MomentumWindowDays(), cloneFetchFrom)
		mLabels, mA, mB = h2h.TrimFloatSeriesFrom(mLabels, mA, mB, from)
	}

	payload := struct {
		RepoA          string     `json:"repoA"`
		RepoB          string     `json:"repoB"`
		ShowMomentum   bool       `json:"showMomentum"`
		CloneLabels    []string   `json:"cloneLabels"`
		ClonesA        []int      `json:"clonesA"`
		ClonesB        []int      `json:"clonesB"`
		ViewLabels     []string   `json:"viewLabels"`
		ViewsA         []int      `json:"viewsA"`
		ViewsB         []int      `json:"viewsB"`
		MomentumLabels []string   `json:"momentumLabels"`
		MomentumA      []*float64 `json:"momentumA"`
		MomentumB      []*float64 `json:"momentumB"`
	}{
		RepoA:          repoA,
		RepoB:          repoB,
		ShowMomentum:   interval.HasMomentum(),
		CloneLabels:    clLabels,
		ClonesA:        clAcounts,
		ClonesB:        clBcounts,
		ViewLabels:     vLabels,
		ViewsA:         vAcounts,
		ViewsB:         vBcounts,
		MomentumLabels: mLabels,
		MomentumA:      mA,
		MomentumB:      mB,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return template.JS("{}"), false
	}
	return template.JS(b), true
}

// h2hChartFromDates returns (chartFrom, cloneFetchFrom). cloneFetchFrom may be earlier for momentum history.
func h2hChartFromDates(now time.Time, interval h2h.Interval) (chartFrom, cloneFetchFrom string) {
	span := interval.ChartSpanDays()
	if span == 0 {
		chartFrom = "2000-01-01"
		return chartFrom, chartFrom
	}
	chartFrom = now.AddDate(0, 0, -span).Format("2006-01-02")
	hist := interval.CloneHistoryDays()
	cloneFetchFrom = now.AddDate(0, 0, -hist).Format("2006-01-02")
	return chartFrom, cloneFetchFrom
}

func trimDayRowsFrom(rows []store.DayRow, from string) []store.DayRow {
	if from == "" {
		return rows
	}
	out := make([]store.DayRow, 0, len(rows))
	for _, r := range rows {
		if r.Date >= from {
			out = append(out, r)
		}
	}
	return out
}

func h2hQueryURL(a, b, interval string) string {
	v := url.Values{}
	if a != "" {
		v.Set("a", a)
	}
	if b != "" {
		v.Set("b", b)
	}
	if interval != "" && interval != string(h2h.Interval7d) {
		v.Set("w", interval)
	}
	q := v.Encode()
	if q == "" {
		return "/h2h"
	}
	return "/h2h?" + q
}
