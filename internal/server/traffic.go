package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hrodrig/gghstats/internal/store"
)

const (
	defaultTrafficDays = 30
	maxTrafficDays     = 3660
)

type repoTrafficResponse struct {
	Name   string         `json:"name"`
	Days   int            `json:"days"`
	From   string         `json:"from"`
	To     string         `json:"to"`
	Clones []store.DayRow `json:"clones"`
	Views  []store.DayRow `json:"views"`
}

func repoFullNameFromRequest(r *http.Request) string {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	if owner == "" || repo == "" {
		return ""
	}
	return owner + "/" + repo
}

// parseTrafficDays interprets the days query parameter (default 30). days=0 means all stored history (UTC).
func parseTrafficDays(raw string) (days int, err error) {
	if raw == "" {
		return defaultTrafficDays, nil
	}
	days, err = strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid days")
	}
	if days < 0 || days > maxTrafficDays {
		return 0, fmt.Errorf("invalid days")
	}
	return days, nil
}

func trafficDateRangeUTC(days int, extentMin string, extentOk bool) (from, to string, err error) {
	to = time.Now().UTC().Format("2006-01-02")
	if days == 0 {
		if !extentOk {
			return to, to, nil
		}
		return extentMin, to, nil
	}
	from = time.Now().UTC().AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	return from, to, nil
}

func handleAPIRepoTraffic(cfg Config) http.HandlerFunc {
	db := cfg.Store
	return func(w http.ResponseWriter, r *http.Request) {
		fullName := repoFullNameFromRequest(r)
		if fullName == "" {
			writeJSONError(w, http.StatusBadRequest, "invalid path")
			return
		}

		days, err := parseTrafficDays(strings.TrimSpace(r.URL.Query().Get("days")))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
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

		extentMin, _, extentOk, err := db.TrafficDateExtentForRepo(fullName)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		from, to, err := trafficDateRangeUTC(days, extentMin, extentOk)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		clones, err := db.ClonesByRange(fullName, from, to)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		views, err := db.ViewsByRange(fullName, from, to)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		resp := repoTrafficResponse{
			Name:   fullName,
			Days:   days,
			From:   from,
			To:     to,
			Clones: clones,
			Views:  views,
		}
		if resp.Clones == nil {
			resp.Clones = []store.DayRow{}
		}
		if resp.Views == nil {
			resp.Views = []store.DayRow{}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		setAPICORS(w, r, cfg.CORSOrigins)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "encode error")
		}
	}
}

func writeJSONError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"error":%q}`, msg)
}
