package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/hrodrig/gghstats/internal/h2h"
	"github.com/hrodrig/gghstats/internal/store"
)

func handleAPIH2H(cfg Config) http.HandlerFunc {
	db := cfg.Store
	return func(w http.ResponseWriter, r *http.Request) {
		rawA := strings.TrimSpace(r.URL.Query().Get("a"))
		rawB := strings.TrimSpace(r.URL.Query().Get("b"))
		interval := h2h.ParseInterval(r.URL.Query().Get("w"))

		if rawA == "" || rawB == "" {
			writeJSONError(w, http.StatusBadRequest, "missing a or b")
			return
		}
		repoA, okA := h2h.ParseRepoFullName(rawA)
		repoB, okB := h2h.ParseRepoFullName(rawB)
		if !okA || !okB {
			writeJSONError(w, http.StatusBadRequest, "invalid repo")
			return
		}
		if repoA == repoB {
			writeJSONError(w, http.StatusBadRequest, "same repo")
			return
		}

		mA, err := h2h.LoadRepoMetrics(db, repoA)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		mB, err := h2h.LoadRepoMetrics(db, repoB)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		if mA == nil || mB == nil {
			writeJSONNotFound(w)
			return
		}
		res, ok := h2h.Compare(mA, mB, interval)
		if !ok {
			writeJSONError(w, http.StatusInternalServerError, "compare failed")
			return
		}

		resp := map[string]interface{}{
			"a":        repoA,
			"b":        repoB,
			"interval": string(interval),
			"result":   res,
		}
		if chart, ok := decodeH2HChartJSON(db, repoA, repoB, interval); ok {
			resp["charts"] = chart
		}
		writeAPIJSON(w, r, cfg, resp)
	}
}

func decodeH2HChartJSON(db *store.Store, repoA, repoB string, interval h2h.Interval) (interface{}, bool) {
	js, ok := buildH2HChartJSON(db, repoA, repoB, interval)
	if !ok {
		return nil, false
	}
	var payload interface{}
	if err := json.Unmarshal([]byte(js), &payload); err != nil {
		return nil, false
	}
	return payload, true
}
