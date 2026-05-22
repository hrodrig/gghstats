package server

import (
	"strings"
	"testing"

	"github.com/hrodrig/gghstats/internal/h2h"
	"github.com/hrodrig/gghstats/internal/i18n"
)

func TestH2HScoreSubline_leaderOnly(t *testing.T) {
	bundle := i18n.MustLoad()
	share := "H2H share (7d)"
	res := h2h.Result{ScoreA: 53, ScoreB: 47, DeltaPct: 11, LeadsA: true}

	a := h2hScoreSubline(bundle, "en", share, res, true)
	b := h2hScoreSubline(bundle, "en", share, res, false)
	if !strings.Contains(a, "margin") || !strings.Contains(a, "11") {
		t.Fatalf("leader A: got %q", a)
	}
	if b != share {
		t.Fatalf("non-leader B: got %q want %q", b, share)
	}
}

func TestH2HScoreSubline_swappedColumns(t *testing.T) {
	bundle := i18n.MustLoad()
	share := "Cuota H2H (7 días)"
	res := h2h.Result{ScoreA: 47, ScoreB: 53, DeltaPct: 11, LeadsA: false}

	a := h2hScoreSubline(bundle, "es", share, res, true)
	b := h2hScoreSubline(bundle, "es", share, res, false)
	if a != share {
		t.Fatalf("loser A: got %q", a)
	}
	if !strings.Contains(b, "ventaja") || !strings.Contains(b, "11") {
		t.Fatalf("leader B: got %q", b)
	}
}

func TestH2HScoreSubline_tiedOrClose(t *testing.T) {
	bundle := i18n.MustLoad()
	share := "H2H share"
	for _, res := range []h2h.Result{
		{ScoreA: 50, ScoreB: 50, DeltaPct: 0, LeadsA: true},
		{ScoreA: 52, ScoreB: 48, DeltaPct: 8, LeadsA: true},
	} {
		if got := h2hScoreSubline(bundle, "en", share, res, true); got != share {
			t.Fatalf("A tied/close: %+v got %q", res, got)
		}
		if got := h2hScoreSubline(bundle, "en", share, res, false); got != share {
			t.Fatalf("B tied/close: %+v got %q", res, got)
		}
	}
}
