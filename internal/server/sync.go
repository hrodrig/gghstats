package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/hrodrig/gghstats/internal/sync"
)

func handleAPISyncStatus(cfg Config) http.HandlerFunc {
	coord := cfg.SyncCoordinator
	return func(w http.ResponseWriter, r *http.Request) {
		if coord == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		setAPICORS(w, r, cfg.CORSOrigins)
		_ = json.NewEncoder(w).Encode(coord.Status())
	}
}

func handleAPISyncStart(cfg Config) http.HandlerFunc {
	coord := cfg.SyncCoordinator
	return func(w http.ResponseWriter, r *http.Request) {
		if coord == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		setAPICORS(w, r, cfg.CORSOrigins)

		repo, err := parseSyncRepoQuery(r.URL.Query().Get("repo"))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		if repo != "" {
			err = coord.StartRepo(repo)
		} else {
			err = coord.Start()
		}
		if err != nil {
			if errors.Is(err, sync.ErrInProgress) {
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error":  "sync_in_progress",
					"status": coord.Status(),
				})
				return
			}
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusAccepted)
		body := map[string]interface{}{"status": "started", "scope": "all"}
		if repo != "" {
			body["scope"] = "repo"
			body["repo"] = repo
		}
		_ = json.NewEncoder(w).Encode(body)
	}
}
