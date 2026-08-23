package server

import (
	"fmt"
	"sort"

	"github.com/hrodrig/gghstats/internal/store"
)

// indexRepoRow adds clone-based ranking information to the homepage view.
// Ranking is calculated before pagination, against the current search/filter set.
type indexRepoRow struct {
	store.RepoSummary
	CloneRank         int
	CloneRankOrdinal  string
	CloneSharePercent float64
}

func rankIndexReposByTotalClones(repos []store.RepoSummary) []indexRepoRow {
	if len(repos) == 0 {
		return nil
	}

	totalClones := 0
	byCloneTotal := append([]store.RepoSummary(nil), repos...)
	for _, repo := range byCloneTotal {
		totalClones += repo.TotalClones
	}
	sort.Slice(byCloneTotal, func(i, j int) bool {
		if byCloneTotal[i].TotalClones == byCloneTotal[j].TotalClones {
			return byCloneTotal[i].Name < byCloneTotal[j].Name
		}
		return byCloneTotal[i].TotalClones > byCloneTotal[j].TotalClones
	})

	ranks := make(map[string]int, len(byCloneTotal))
	for i, repo := range byCloneTotal {
		ranks[repo.Name] = i + 1
	}

	rows := make([]indexRepoRow, len(repos))
	for i, repo := range repos {
		rank := ranks[repo.Name]
		share := 0.0
		if totalClones > 0 {
			share = float64(repo.TotalClones) * 100 / float64(totalClones)
		}
		rows[i] = indexRepoRow{
			RepoSummary:       repo,
			CloneRank:         rank,
			CloneRankOrdinal:  ordinal(rank),
			CloneSharePercent: share,
		}
	}
	return rows
}

func indexRepoRowsPageSlice(repos []indexRepoRow, page, perPage int) (start, end int, pageSlice []indexRepoRow) {
	total := len(repos)
	start = (page - 1) * perPage
	if start > total {
		start = total
	}
	end = start + perPage
	if end > total {
		end = total
	}
	return start, end, repos[start:end]
}

func ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}
