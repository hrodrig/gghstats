package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hrodrig/gghstats/internal/h2h"
	"github.com/hrodrig/gghstats/internal/store"
	"github.com/hrodrig/gghstats/internal/version"
)

type h2hRepoOption struct {
	Name string
}

type h2hPageData struct {
	Repos              []h2hRepoOption
	RepoA              string
	RepoB              string
	Interval           string
	IntervalLabel      string
	ShowMomentumChart  bool
	ChartSpanLabel     string
	MomentumChartLabel string
	Error              string
	Compared           bool
	Result             h2h.Result
	LeadsLabel         string
	ChartJSON          template.JS
	HasCharts          bool
	FormAction         string
}

func handleH2HPage(cfg Config, db *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/h2h" {
			writeBrutalistNotFound(w, tmpl, cfg, "Not found", "Page not found", r.URL.Path,
				"The requested page does not exist, or the URL may be incorrect.")
			return
		}

		repos, err := db.ListRepos("name", "asc")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rawA := r.URL.Query().Get("a")
		rawB := r.URL.Query().Get("b")
		interval := h2h.ParseInterval(r.URL.Query().Get("w"))
		data := newH2HPageData(repos, rawA, rawB, interval)

		if err := applyH2HComparison(&data, db, rawA, rawB, interval); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		content := executeTemplate(tmpl, "h2h", data)
		renderLayout(w, tmpl, cfg, layoutData{
			Title:   "Head to Head",
			Version: version.Version,
			Breadcrumbs: []breadcrumb{
				{Label: "Home", URL: "/"},
				{Label: "H2H", URL: ""},
			},
			Content: content,
		})
	}
}

func newH2HPageData(repos []store.RepoSummary, rawA, rawB string, interval h2h.Interval) h2hPageData {
	opts := make([]h2hRepoOption, len(repos))
	for i, rp := range repos {
		opts[i] = h2hRepoOption{Name: rp.Name}
	}
	return h2hPageData{
		Repos:              opts,
		RepoA:              strings.TrimSpace(rawA),
		RepoB:              strings.TrimSpace(rawB),
		Interval:           string(interval),
		IntervalLabel:      interval.Label(),
		ShowMomentumChart:  interval.HasMomentum(),
		ChartSpanLabel:     h2hChartSpanLabel(interval),
		MomentumChartLabel: h2hMomentumChartLabel(interval),
		FormAction:         "/h2h",
	}
}

// applyH2HComparison fills comparison fields when query params request a pair. Returns an error for load failures.
func applyH2HComparison(data *h2hPageData, db *store.Store, rawA, rawB string, interval h2h.Interval) error {
	if rawA == "" && rawB == "" {
		return nil
	}
	repoA, okA := h2h.ParseRepoFullName(rawA)
	repoB, okB := h2h.ParseRepoFullName(rawB)
	if !okA || !okB {
		data.Error = "Enter valid owner/repo names for both repositories."
		return nil
	}
	if repoA == repoB {
		data.Error = "Choose two different repositories."
		return nil
	}
	return populateH2HComparison(data, db, repoA, repoB, interval)
}

func populateH2HComparison(data *h2hPageData, db *store.Store, repoA, repoB string, interval h2h.Interval) error {
	mA, err := h2h.LoadRepoMetrics(db, repoA)
	if err != nil {
		return err
	}
	mB, err := h2h.LoadRepoMetrics(db, repoB)
	if err != nil {
		return err
	}
	if mA == nil || mB == nil {
		data.Error = "One or both repositories were not found in the database."
		return nil
	}
	res, ok := h2h.Compare(mA, mB, interval)
	if !ok {
		return nil
	}
	data.Compared = true
	data.Result = res
	data.RepoA = repoA
	data.RepoB = repoB
	if res.LeadsA {
		data.LeadsLabel = repoA
	} else {
		data.LeadsLabel = repoB
	}
	if chartJS, ok := buildH2HChartJSON(db, repoA, repoB, interval); ok {
		data.ChartJSON = chartJS
		data.HasCharts = true
	}
	return nil
}

func h2hChartSpanLabel(interval h2h.Interval) string {
	switch interval {
	case h2h.Interval30d:
		return "30 days"
	case h2h.IntervalTotal:
		return "all time"
	default:
		return "7 days"
	}
}

func h2hMomentumChartLabel(interval h2h.Interval) string {
	switch interval {
	case h2h.Interval30d:
		return "Momentum (30d vs prev 30d)"
	default:
		return "Momentum (7d vs prev 7d)"
	}
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
