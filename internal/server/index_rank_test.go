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
	})

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

func TestOrdinal(t *testing.T) {
	for input, want := range map[int]string{1: "1st", 2: "2nd", 3: "3rd", 4: "4th", 11: "11th", 12: "12th", 13: "13th", 21: "21st"} {
		if got := ordinal(input); got != want {
			t.Errorf("ordinal(%d) = %q, want %q", input, got, want)
		}
	}
}
