package metrics

import (
	"strings"
	"testing"

	"github.com/hrodrig/gghstats/internal/store"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestSplitRepo(t *testing.T) {
	owner, repo, ok := splitRepo("hrodrig/gghstats")
	if !ok || owner != "hrodrig" || repo != "gghstats" {
		t.Fatalf("splitRepo = %q %q %v", owner, repo, ok)
	}
	if _, _, ok := splitRepo("invalid"); ok {
		t.Fatal("expected invalid name")
	}
}

func TestRefreshPerRepoGauges(t *testing.T) {
	reg := prometheus.NewRegistry()
	d := RegisterDomain(reg, DomainConfig{
		Filter:         "*",
		PerRepoEnabled: true,
		ListRepos: func() ([]store.RepoSummary, error) {
			return []store.RepoSummary{
				{Name: "o/r", Stars: 3, Forks: 1, TotalClones: 10, TotalViews: 5, Clones1d: 1, Clones7d: 4, Clones30d: 9},
			}, nil
		},
	})
	d.refreshPerRepoGauges()

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	text := ""
	for _, mf := range mfs {
		text += mf.GetName() + "\n"
	}
	for _, name := range []string{
		"gghstats_repo_clones_1d",
		"gghstats_repo_clones_7d",
		"gghstats_repo_clones_30d",
	} {
		if !strings.Contains(text, name) {
			t.Errorf("missing metric family %s", name)
		}
	}
}

func TestSplitRepoKey(t *testing.T) {
	owner, repo, ok := splitRepoKey("o\x00r")
	if !ok || owner != "o" || repo != "r" {
		t.Fatalf("splitRepoKey = %q %q %v", owner, repo, ok)
	}
	if _, _, ok := splitRepoKey("bad"); ok {
		t.Fatal("expected invalid key")
	}
}

func TestRefreshPerRepoRemovesStaleLabels(t *testing.T) {
	reg := prometheus.NewRegistry()
	list := []store.RepoSummary{
		{Name: "keep/a", Stars: 1},
		{Name: "gone/b", Stars: 2},
	}
	d := RegisterDomain(reg, DomainConfig{
		PerRepoEnabled: true,
		ListRepos: func() ([]store.RepoSummary, error) {
			return list, nil
		},
	})
	d.refreshPerRepoGauges()

	list = []store.RepoSummary{{Name: "keep/a", Stars: 3}}
	d.refreshPerRepoGauges()

	if _, ok := d.lastRepoLabels[repoLabelKey("gone", "b")]; ok {
		t.Fatal("stale repo should be removed from lastRepoLabels")
	}
	keep, err := d.repoStars.GetMetricWithLabelValues("keep", "a")
	if err != nil {
		t.Fatal(err)
	}
	if v := testutil.ToFloat64(keep); v != 3 {
		t.Fatalf("keep/a stars = %v, want 3", v)
	}
}

func TestRefreshPerRepoSkipsInvalidNames(t *testing.T) {
	reg := prometheus.NewRegistry()
	d := RegisterDomain(reg, DomainConfig{
		PerRepoEnabled: true,
		ListRepos: func() ([]store.RepoSummary, error) {
			return []store.RepoSummary{{Name: "not-a-repo", Stars: 1}}, nil
		},
	})
	d.refreshPerRepoGauges()
	if len(d.lastRepoLabels) != 0 {
		t.Fatalf("lastRepoLabels = %v, want empty", d.lastRepoLabels)
	}
}
