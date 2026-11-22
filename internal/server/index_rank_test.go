package server

import (
	"testing"

	"github.com/hrodrig/gghstats/internal/store"
)

func TestRankIndexReposByTotalClones(t *testing.T) {
	rows := rankIndexReposByTotalClones([]store.RepoSummary{
		{Name: "owner/two", TotalClones: 20},
		{Name: "owner/zero", TotalClones: 0},
		{Name: "owner/one", TotalClones: 80},
	}, "en")

	if got := rows[0]; got.CloneRank != 2 || got.CloneRankOrdinal != "2nd" || got.CloneSharePercent != 20 {
		t.Fatalf("row 0 = %#v, want second at 20%%", got)
	}
	if got := rows[1]; got.CloneRank != 3 || got.CloneRankOrdinal != "3rd" || got.CloneSharePercent != 0 {
		t.Fatalf("row 1 = %#v, want third at 0%%", got)
	}
	if got := rows[2]; got.CloneRank != 1 || got.CloneRankOrdinal != "1st" || got.CloneSharePercent != 80 {
		t.Fatalf("row 2 = %#v, want first at 80%%", got)
	}
}

func TestRankIndexReposByTotalClonesTiedShareCompetitionRank(t *testing.T) {
	// Display order may put the alphabetically later repo first (e.g. higher unique
	// clones). Competition ranks must still match so ordinals never invert (22nd above 21st).
	rows := rankIndexReposByTotalClones([]store.RepoSummary{
		{Name: "hrodrig/kui-selfhosted", TotalClones: 359, CloneUniques: 209},
		{Name: "hrodrig/gfire", TotalClones: 359, CloneUniques: 192},
		{Name: "hrodrig/ahead", TotalClones: 1000, CloneUniques: 50},
	}, "en")

	if got := rows[0]; got.CloneRank != 2 || got.CloneRankOrdinal != "2nd" {
		t.Fatalf("tied row 0 = rank %d %q, want 2 / 2nd", got.CloneRank, got.CloneRankOrdinal)
	}
	if got := rows[1]; got.CloneRank != 2 || got.CloneRankOrdinal != "2nd" {
		t.Fatalf("tied row 1 = rank %d %q, want 2 / 2nd (same competition rank)", got.CloneRank, got.CloneRankOrdinal)
	}
	if got := rows[2]; got.CloneRank != 1 || got.CloneRankOrdinal != "1st" {
		t.Fatalf("leader = rank %d %q, want 1 / 1st", got.CloneRank, got.CloneRankOrdinal)
	}
}

func TestOrdinal(t *testing.T) {
	for _, tc := range []struct {
		n      int
		locale string
		want   string
	}{
		{1, "en", "1st"}, {2, "en", "2nd"}, {3, "en", "3rd"}, {4, "en", "4th"},
		{11, "en", "11th"}, {12, "en", "12th"}, {13, "en", "13th"}, {21, "en", "21st"},
		{1, "es", "1.º"}, {2, "es", "2.º"}, {21, "es", "21.º"},
		{1, "de", "#1"}, {15, "fr", "#15"}, {3, "pt-br", "#3"},
	} {
		if got := ordinal(tc.n, tc.locale); got != tc.want {
			t.Errorf("ordinal(%d, %q) = %q, want %q", tc.n, tc.locale, got, tc.want)
		}
	}
}
