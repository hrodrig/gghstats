package server

import (
	"encoding/json"
	"net/http"

	"github.com/hrodrig/gghstats/internal/store"
)

func handleAPIIndexClonesChart(cfg Config) http.HandlerFunc {
	db := cfg.Store
	return func(w http.ResponseWriter, r *http.Request) {
		sort, dir, query, _, _, _ := parseAPIReposQuery(r)
		repos, err := loadFilteredIndexRepos(db, sort, dir, query)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		count, js, err := buildIndexListClonesChartPayload(db, repoNamesFromSummaries(repos))
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		var series interface{}
		if err := json.Unmarshal([]byte(js), &series); err != nil {
			series = []store.DayRow{}
		}
		writeAPIJSON(w, r, cfg, map[string]interface{}{
			"count":  count,
			"series": series,
			"sort":   sort,
			"dir":    dir,
			"q":      query,
		})
	}
}
