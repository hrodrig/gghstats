package h2h

import (
	"strings"
	"testing"
)

func TestParseRepoFullName(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"hrodrig/gghstats", "hrodrig/gghstats", true},
		{"  hrodrig/pgwd  ", "hrodrig/pgwd", true},
		{"", "", false},
		{"nope", "", false},
		{"a/b/c", "", false},
	}
	for _, tt := range tests {
		got, ok := ParseRepoFullName(tt.in)
		if ok != tt.ok || got != tt.want {
			t.Errorf("ParseRepoFullName(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestCompare_clearWinner(t *testing.T) {
	a := &RepoMetrics{Name: "a/r", Clones7d: 100, Clones30d: 500, Views7d: 50, Momentum7d: 0.5}
	b := &RepoMetrics{Name: "b/r", Clones7d: 10, Clones30d: 50, Views7d: 5, Momentum7d: 0.1}
	res, ok := Compare(a, b, Interval7d)
	if !ok {
		t.Fatal("expected ok")
	}
	if !res.LeadsA {
		t.Error("expected A to lead")
	}
	if res.ScoreA <= res.ScoreB {
		t.Errorf("scoreA=%d scoreB=%d", res.ScoreA, res.ScoreB)
	}
	if !strings.Contains(res.Suggest.Rationale, "winner") {
		t.Errorf("rationale = %q, want winner mention", res.Suggest.Rationale)
	}
}

func TestCompare_tooClose(t *testing.T) {
	m := &RepoMetrics{Name: "x/r", Clones7d: 100, Clones30d: 200, Views7d: 40, Momentum7d: 0.1}
	res, ok := Compare(m, m, Interval7d)
	if !ok {
		t.Fatal("expected ok")
	}
	if res.ScoreA != 50 || res.ScoreB != 50 {
		t.Errorf("scores = %d/%d, want 50/50 for identical repos", res.ScoreA, res.ScoreB)
	}
	if !strings.Contains(res.Suggest.Rationale, "too close") {
		t.Errorf("rationale = %q, want too close", res.Suggest.Rationale)
	}
}

func TestCompare_nilMetrics(t *testing.T) {
	if _, ok := Compare(nil, &RepoMetrics{Name: "a/r"}, Interval7d); ok {
		t.Error("expected false with nil A")
	}
	if _, ok := Compare(&RepoMetrics{Name: "a/r"}, nil, Interval7d); ok {
		t.Error("expected false with nil B")
	}
}

func TestCompare_30dInterval(t *testing.T) {
	a := &RepoMetrics{Name: "a/r", Clones30d: 300, Views30d: 90, Momentum30d: 0.2}
	b := &RepoMetrics{Name: "b/r", Clones30d: 100, Views30d: 30, Momentum30d: 0.1}
	res, ok := Compare(a, b, Interval30d)
	if !ok || !res.LeadsA {
		t.Fatalf("compare 30d: ok=%v leadsA=%v scores=%d/%d", ok, res.LeadsA, res.ScoreA, res.ScoreB)
	}
}

func TestSharePair_andDeltaPct(t *testing.T) {
	a, b := sharePair(0, 0)
	if a != 0.5 || b != 0.5 {
		t.Errorf("zero pair = %v/%v", a, b)
	}
	a, b = sharePair(-1, 10)
	if a != 0 || b != 1 {
		t.Errorf("negative clamp = %v/%v", a, b)
	}
	if deltaPct(0, 0) != 0 {
		t.Errorf("delta zero leader = %v", deltaPct(0, 0))
	}
	if deltaPct(62, 38) < 35 || deltaPct(62, 38) > 45 {
		t.Errorf("delta 62/38 = %v", deltaPct(62, 38))
	}
}

func TestCompare_proportionalNotBinary(t *testing.T) {
	// Realistic pair: A leads every row but not 100× — scores must not be 0/100.
	a := &RepoMetrics{Name: "hrodrig/gghstats", Clones7d: 358, Views7d: 92, Momentum7d: 4.26}
	b := &RepoMetrics{Name: "hrodrig/pgwd", Clones7d: 209, Views7d: 69, Momentum7d: 2.32}
	res, ok := Compare(a, b, Interval7d)
	if !ok {
		t.Fatal("expected ok")
	}
	if res.ScoreA <= res.ScoreB {
		t.Fatalf("scoreA=%d scoreB=%d, want A ahead", res.ScoreA, res.ScoreB)
	}
	if res.ScoreA >= 95 || res.ScoreB <= 5 {
		t.Errorf("scores too extreme %d/%d; want proportional split", res.ScoreA, res.ScoreB)
	}
	if res.DeltaPct > 80 {
		t.Errorf("delta=%.0f, want a moderate lead gap", res.DeltaPct)
	}
	if res.ScoreA+res.ScoreB < 98 || res.ScoreA+res.ScoreB > 102 {
		t.Errorf("scores sum %d, want ~100", res.ScoreA+res.ScoreB)
	}
}
