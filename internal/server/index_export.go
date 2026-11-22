package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

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

func buildIndexJSONLExportRow(db *store.Store, repo indexRepoRow) (indexJSONLExportRow, error) {
	clones, err := db.ClonesByRange(repo.Name, allHistoryStart, allHistoryEnd)
	if err != nil {
		return indexJSONLExportRow{}, err
	}
	views, err := db.ViewsByRange(repo.Name, allHistoryStart, allHistoryEnd)
	if err != nil {
		return indexJSONLExportRow{}, err
	}
	referrers, err := db.ReferrersByRange(repo.Name, allHistoryStart, allHistoryEnd)
	if err != nil {
		return indexJSONLExportRow{}, err
	}
	paths, err := db.PathsByRange(repo.Name, allHistoryStart, allHistoryEnd)
	if err != nil {
		return indexJSONLExportRow{}, err
	}
	stars, err := db.StarsByRepo(repo.Name)
	if err != nil {
		return indexJSONLExportRow{}, err
	}
	return indexJSONLExportRow{
		Repo:              repo.RepoSummary,
		CloneRank:         repo.CloneRank,
		CloneSharePercent: repo.CloneSharePercent,
		Clones:            nonNilDayRows(clones),
		Views:             nonNilDayRows(views),
		Referrers:         nonNilReferrerRows(referrers),
		Paths:             nonNilPathRows(paths),
		Stars:             nonNilStarRows(stars),
	}, nil
}

func handleIndexJSONLExport(cfg Config) http.HandlerFunc {
	db := cfg.Store
	return func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		repos, err := loadFilteredIndexRepos(db, cfg.ReportVisibility, "total_clones", "desc", query)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		rankedRepos := rankIndexReposByTotalClones(repos, localeFromRequest(r, cfg))

		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, indexJSONLExportFilename(time.Now().UTC())))
		w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
		encoder := json.NewEncoder(w)
		for _, repo := range rankedRepos {
			row, err := buildIndexJSONLExportRow(db, repo)
			if err != nil {
				slog.Error("build JSONL export row", "repo", repo.Name, "error", err)
				// Headers may already be sent; stop without rewriting status.
				return
			}
			if err := encoder.Encode(row); err != nil {
				slog.Error("write JSONL export", "repo", repo.Name, "error", err)
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

// indexJSONLExportFilename returns gghstats-export-YYYYMMDD-HHMM.jsonl in UTC (#23).
func indexJSONLExportFilename(t time.Time) string {
	return "gghstats-export-" + t.UTC().Format("20060102-1504") + ".jsonl"
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
