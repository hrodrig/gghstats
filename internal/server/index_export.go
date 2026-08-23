package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/hrodrig/gghstats/internal/store"
)

const (
	allHistoryStart = "0001-01-01"
	allHistoryEnd   = "9999-12-31"
)

// indexJSONLExportRow is one repository and all of the data the dashboard stores for it.
// A JSONL file has one of these objects per line, making it convenient to stream or import.
type indexJSONLExportRow struct {
	Repo              store.RepoSummary   `json:"repo"`
	CloneRank         int                 `json:"clone_rank"`
	CloneSharePercent float64             `json:"clone_share_percent"`
	Clones            []store.DayRow      `json:"clones"`
	Views             []store.DayRow      `json:"views"`
	Referrers         []store.ReferrerRow `json:"referrers"`
	Paths             []store.PathRow     `json:"paths"`
	Stars             []store.StarRow     `json:"stars"`
}

func buildIndexJSONLExportRows(db *store.Store, repos []store.RepoSummary) ([]indexJSONLExportRow, error) {
	rankedRepos := rankIndexReposByTotalClones(repos)
	rows := make([]indexJSONLExportRow, 0, len(rankedRepos))
	for _, repo := range rankedRepos {
		clones, err := db.ClonesByRange(repo.Name, allHistoryStart, allHistoryEnd)
		if err != nil {
			return nil, err
		}
		views, err := db.ViewsByRange(repo.Name, allHistoryStart, allHistoryEnd)
		if err != nil {
			return nil, err
		}
		referrers, err := db.ReferrersByRange(repo.Name, allHistoryStart, allHistoryEnd)
		if err != nil {
			return nil, err
		}
		paths, err := db.PathsByRange(repo.Name, allHistoryStart, allHistoryEnd)
		if err != nil {
			return nil, err
		}
		stars, err := db.StarsByRepo(repo.Name)
		if err != nil {
			return nil, err
		}
		rows = append(rows, indexJSONLExportRow{
			Repo:              repo.RepoSummary,
			CloneRank:         repo.CloneRank,
			CloneSharePercent: repo.CloneSharePercent,
			Clones:            nonNilDayRows(clones),
			Views:             nonNilDayRows(views),
			Referrers:         nonNilReferrerRows(referrers),
			Paths:             nonNilPathRows(paths),
			Stars:             nonNilStarRows(stars),
		})
	}
	return rows, nil
}

func handleIndexJSONLExport(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		repos, err := loadFilteredIndexRepos(db, "total_clones", "desc", query)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		rows, err := buildIndexJSONLExportRows(db, repos)
		if err != nil {
			slog.Error("build JSONL export", "error", err)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Disposition", `attachment; filename="gghstats-export.jsonl"`)
		w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
		encoder := json.NewEncoder(w)
		for _, row := range rows {
			if err := encoder.Encode(row); err != nil {
				slog.Error("write JSONL export", "error", err)
				return
			}
		}
	}
}

func nonNilDayRows(rows []store.DayRow) []store.DayRow {
	if rows == nil {
		return []store.DayRow{}
	}
	return rows
}

func nonNilReferrerRows(rows []store.ReferrerRow) []store.ReferrerRow {
	if rows == nil {
		return []store.ReferrerRow{}
	}
	return rows
}

func nonNilPathRows(rows []store.PathRow) []store.PathRow {
	if rows == nil {
		return []store.PathRow{}
	}
	return rows
}

func nonNilStarRows(rows []store.StarRow) []store.StarRow {
	if rows == nil {
		return []store.StarRow{}
	}
	return rows
}
