package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/hrodrig/gghstats/internal/h2h"
	"github.com/hrodrig/gghstats/internal/store"
)

func handleAPIRepo(cfg Config) http.HandlerFunc {
	db := cfg.Store
	return func(w http.ResponseWriter, r *http.Request) {
		fullName := repoFullNameFromRequest(r)
		if fullName == "" {
			writeJSONError(w, http.StatusBadRequest, "invalid path")
			return
		}
		summary, err := db.RepoByName(fullName)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		if summary == nil {
			writeJSONNotFound(w)
			return
		}

		resp := map[string]interface{}{
			"repo": summary,
		}
		if m, err := h2h.LoadRepoMetrics(db, fullName); err != nil {
			slog.Warn("api repo momentum", "repo", fullName, "error", err)
		} else if m != nil {
			resp["momentum_7d"] = m.Momentum7d
			resp["momentum_30d"] = m.Momentum30d
			resp["momentum_7d_pct"] = h2h.FormatMomentumPct(m.Momentum7d)
			resp["momentum_30d_pct"] = h2h.FormatMomentumPct(m.Momentum30d)
		}

		writeAPIJSON(w, r, cfg, resp)
	}
}

func handleAPIRepoStars(cfg Config) http.HandlerFunc {
	db := cfg.Store
	return func(w http.ResponseWriter, r *http.Request) {
		fullName := repoFullNameFromRequest(r)
		if fullName == "" {
			writeJSONError(w, http.StatusBadRequest, "invalid path")
			return
		}
		summary, err := db.RepoByName(fullName)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		if summary == nil {
			writeJSONNotFound(w)
			return
		}
		stars, err := db.StarsByRepo(fullName)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		if stars == nil {
			stars = []store.StarRow{}
		}
		writeAPIJSON(w, r, cfg, map[string]interface{}{
			"name":  fullName,
			"stars": stars,
		})
	}
}

func handleAPIRepoPopular(cfg Config) http.HandlerFunc {
	db := cfg.Store
	return func(w http.ResponseWriter, r *http.Request) {
		fullName := repoFullNameFromRequest(r)
		if fullName == "" {
			writeJSONError(w, http.StatusBadRequest, "invalid path")
			return
		}
		summary, err := db.RepoByName(fullName)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		if summary == nil {
			writeJSONNotFound(w)
			return
		}
		const windowDays = 14
		referrers, err := db.PopularReferrers(fullName, windowDays)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		paths, err := db.PopularPaths(fullName, windowDays)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		if referrers == nil {
			referrers = []store.PopularItem{}
		}
		if paths == nil {
			paths = []store.PopularItem{}
		}
		writeAPIJSON(w, r, cfg, map[string]interface{}{
			"name":      fullName,
			"days":      windowDays,
			"referrers": referrers,
			"paths":     paths,
		})
	}
}

func writeAPIJSON(w http.ResponseWriter, r *http.Request, cfg Config, resp interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	setAPICORS(w, r, cfg.CORSOrigins)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("encode api json", "error", err)
	}
}
