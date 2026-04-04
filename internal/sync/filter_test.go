package sync

import (
	"testing"

	"github.com/hrodrig/gghstats/internal/github"
)

func makeRepos(names ...string) []github.Repo {
	var repos []github.Repo
	for _, n := range names {
		repos = append(repos, github.Repo{FullName: n})
	}
	return repos
}

func repoNames(repos []github.Repo) []string {
	var names []string
	for _, r := range repos {
		names = append(names, r.FullName)
	}
	return names
}

func TestFilterAll(t *testing.T) {
	repos := makeRepos("a/1", "a/2", "b/1")
	got := applyFilter(repos, "*")
	if len(got) != 3 {
		t.Errorf("got %d, want 3", len(got))
	}
}

func TestFilterSpecificRepos(t *testing.T) {
	repos := makeRepos("a/1", "a/2", "b/1")
	got := applyFilter(repos, "a/1,b/1")
	if len(got) != 2 {
		t.Errorf("got %v, want [a/1 b/1]", repoNames(got))
	}
}

func TestFilterWildcard(t *testing.T) {
	repos := makeRepos("a/1", "a/2", "b/1")
	got := applyFilter(repos, "a/*")
	if len(got) != 2 {
		t.Errorf("got %v, want [a/1 a/2]", repoNames(got))
	}
}

func TestFilterExclude(t *testing.T) {
	repos := makeRepos("a/1", "a/2", "b/1")
	got := applyFilter(repos, "*,!a/2")
	if len(got) != 2 {
		t.Errorf("got %v, want [a/1 b/1]", repoNames(got))
	}
}

func TestFilterExcludeFork(t *testing.T) {
	repos := []github.Repo{
		{FullName: "a/1", Fork: false},
		{FullName: "a/2", Fork: true},
		{FullName: "b/1", Fork: false},
	}
	got := applyFilter(repos, "*,!fork")
	if len(got) != 2 {
		t.Errorf("got %v, want [a/1 b/1]", repoNames(got))
	}
}

func TestFilterExcludeArchived(t *testing.T) {
	repos := []github.Repo{
		{FullName: "a/1"},
		{FullName: "a/2", Archived: true},
	}
	got := applyFilter(repos, "*,!archived")
	if len(got) != 1 {
		t.Errorf("got %v, want [a/1]", repoNames(got))
	}
}

func TestFilterCombined(t *testing.T) {
	repos := []github.Repo{
		{FullName: "x/a"},
		{FullName: "x/b", Fork: true},
		{FullName: "y/c"},
	}
	got := applyFilter(repos, "x/*,!fork")
	if len(got) != 1 || got[0].FullName != "x/a" {
		t.Errorf("got %v, want [x/a]", repoNames(got))
	}
}

func TestFilterExcludeWildcardPrefix(t *testing.T) {
	repos := makeRepos("acme/a", "acme/b", "other/z")
	got := applyFilter(repos, "*,!acme/*")
	if len(got) != 1 || got[0].FullName != "other/z" {
		t.Errorf("got %v, want [other/z]", repoNames(got))
	}
}
