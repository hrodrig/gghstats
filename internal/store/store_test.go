package store

import (
	"os"
	"path/filepath"
	"testing"
)

func tempDB(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenCreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "deep", "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	s.Close()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("database file not created")
	}
}

func TestVersionedMigrations(t *testing.T) {
	s := tempDB(t)

	var ver int
	if err := s.DB().QueryRow("PRAGMA user_version").Scan(&ver); err != nil {
		t.Fatal(err)
	}
	if ver != 3 {
		t.Errorf("user_version = %d, want 3", ver)
	}

	// repos table should exist
	var count int
	if err := s.DB().QueryRow("SELECT COUNT(*) FROM repos").Scan(&count); err != nil {
		t.Fatalf("repos table missing: %v", err)
	}

	var col string
	if err := s.DB().QueryRow(`SELECT name FROM pragma_table_info('repos') WHERE name = 'parent_full_name'`).Scan(&col); err != nil {
		t.Fatalf("parent_full_name column: %v", err)
	}
	if col != "parent_full_name" {
		t.Errorf("repos.parent_full_name column missing, got %q", col)
	}

	// stars table should exist
	if err := s.DB().QueryRow("SELECT COUNT(*) FROM stars").Scan(&count); err != nil {
		t.Fatalf("stars table missing: %v", err)
	}
}

func TestReOpenPreservesMigrationVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	s1, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s1.UpsertView("r", "2026-01-01", 10, 5)
	s1.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()

	rows, err := s2.ViewsByRange("r", "2026-01-01", "2026-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Count != 10 {
		t.Errorf("data not preserved across reopen")
	}
}

func TestUpsertAndQueryViews(t *testing.T) {
	s := tempDB(t)

	if err := s.UpsertView("owner/repo", "2026-03-20", 10, 5); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertView("owner/repo", "2026-03-21", 20, 8); err != nil {
		t.Fatal(err)
	}
	// Upsert same date with lower value — MAX should keep old
	if err := s.UpsertView("owner/repo", "2026-03-20", 8, 4); err != nil {
		t.Fatal(err)
	}

	rows, err := s.ViewsByRange("owner/repo", "2026-03-19", "2026-03-22")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0].Count != 10 {
		t.Errorf("rows[0].Count = %d, want 10 (MAX kept)", rows[0].Count)
	}

	// Upsert same date with higher value — should update
	if err := s.UpsertView("owner/repo", "2026-03-20", 15, 6); err != nil {
		t.Fatal(err)
	}
	rows, _ = s.ViewsByRange("owner/repo", "2026-03-20", "2026-03-20")
	if rows[0].Count != 15 {
		t.Errorf("rows[0].Count = %d, want 15 (MAX updated)", rows[0].Count)
	}
}

func TestUpsertAndQueryClones(t *testing.T) {
	s := tempDB(t)

	if err := s.UpsertClone("r", "2026-03-20", 50, 20); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertClone("r", "2026-03-21", 30, 10); err != nil {
		t.Fatal(err)
	}

	rows, err := s.ClonesByRange("r", "2026-03-20", "2026-03-21")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[1].Count != 30 {
		t.Errorf("rows[1].Count = %d, want 30", rows[1].Count)
	}
}

func TestUpsertAndQueryReferrers(t *testing.T) {
	s := tempDB(t)

	if err := s.UpsertReferrer("r", "2026-03-20", "google.com", 40, 10); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertReferrer("r", "2026-03-20", "github.com", 20, 5); err != nil {
		t.Fatal(err)
	}
	// Upsert same — MAX keeps highest
	if err := s.UpsertReferrer("r", "2026-03-20", "google.com", 45, 12); err != nil {
		t.Fatal(err)
	}

	rows, err := s.ReferrersByRange("r", "2026-03-20", "2026-03-20")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0].Count != 45 {
		t.Errorf("google.com count = %d, want 45", rows[0].Count)
	}
}

func TestUpsertAndQueryPaths(t *testing.T) {
	s := tempDB(t)

	if err := s.UpsertPath("r", "2026-03-20", "/repo", "Repo", 100, 20); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertPath("r", "2026-03-20", "/repo/issues", "Issues", 50, 10); err != nil {
		t.Fatal(err)
	}

	rows, err := s.PathsByRange("r", "2026-03-20", "2026-03-20")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0].Path != "/repo" {
		t.Errorf("first path = %q, want /repo", rows[0].Path)
	}
}

func TestListReposSortByNameAsc(t *testing.T) {
	s := tempDB(t)
	s.UpsertRepo("z/last", "", 1, 0, 0, 0, 0, false, false, "")
	s.UpsertRepo("a/first", "", 99, 0, 0, 0, 0, false, false, "")

	repos, err := s.ListRepos("name", "asc")
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 || repos[0].Name != "a/first" || repos[1].Name != "z/last" {
		t.Fatalf("order: %+v", repos)
	}
}

func TestListReposInvalidSortUsesDefault(t *testing.T) {
	s := tempDB(t)
	s.UpsertRepo("r1", "", 0, 0, 0, 0, 0, false, false, "")
	s.UpsertRepo("r2", "", 0, 0, 0, 0, 0, false, false, "")
	s.UpsertView("r1", "2026-01-01", 100, 50)
	s.UpsertView("r2", "2026-01-01", 10, 5)

	repos, err := s.ListRepos("not-a-valid-key", "desc")
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("want 2 repos, got %d", len(repos))
	}
	// Default is total_views desc — r1 should rank first.
	if repos[0].Name != "r1" {
		t.Errorf("first = %q, want r1 (higher views)", repos[0].Name)
	}
}

func TestPopularPathsAggregated(t *testing.T) {
	s := tempDB(t)
	// Same monotonic shape as TestUpdateDeltas referrers — deltas sum to cumulative net new traffic.
	if err := s.UpsertPath("r", "2026-03-20", "/doc", "Doc", 40, 10); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertPath("r", "2026-03-21", "/doc", "Doc", 50, 15); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertPath("r", "2026-03-22", "/doc", "Doc", 55, 16); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateDeltas(); err != nil {
		t.Fatal(err)
	}

	items, err := s.PopularPaths("r", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].Name != "/doc" || items[0].Count != 55 {
		t.Errorf("item = %+v, want path /doc count_delta sum 55", items[0])
	}
	if items[0].Uniques != 16 {
		t.Errorf("uniques_delta sum = %d, want 16", items[0].Uniques)
	}
}

func TestUpsertRepoAndList(t *testing.T) {
	s := tempDB(t)

	if err := s.UpsertRepo("hrodrig/pgwd", "Postgres Watch Dog", 178, 5, 178, 3, 1, false, false, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertRepo("hrodrig/gghstats", "GitHub stats", 10, 0, 10, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	s.UpsertView("hrodrig/pgwd", "2026-03-20", 100, 50)
	s.UpsertClone("hrodrig/pgwd", "2026-03-20", 30, 10)

	repos, err := s.ListRepos("stars", "desc")
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("got %d repos, want 2", len(repos))
	}
	if repos[0].Name != "hrodrig/pgwd" {
		t.Errorf("first repo = %q, want hrodrig/pgwd (highest stars)", repos[0].Name)
	}
	if repos[0].TotalViews != 100 {
		t.Errorf("total views = %d, want 100", repos[0].TotalViews)
	}
}

func TestRepoByName(t *testing.T) {
	s := tempDB(t)

	s.UpsertRepo("a/b", "desc", 5, 1, 5, 0, 0, false, false, "")
	s.UpsertView("a/b", "2026-01-01", 10, 5)

	r, err := s.RepoByName("a/b")
	if err != nil {
		t.Fatal(err)
	}
	if r == nil {
		t.Fatal("expected repo, got nil")
	}
	if r.Stars != 5 {
		t.Errorf("stars = %d, want 5", r.Stars)
	}

	r2, err := s.RepoByName("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if r2 != nil {
		t.Error("expected nil for nonexistent repo")
	}
}

func TestUpsertForkWithParent(t *testing.T) {
	s := tempDB(t)

	if err := s.UpsertRepo("you/book", "fork desc", 1, 0, 1, 0, 0, true, false, "rust-lang/book"); err != nil {
		t.Fatal(err)
	}
	r, err := s.RepoByName("you/book")
	if err != nil {
		t.Fatal(err)
	}
	if r == nil {
		t.Fatal("expected repo")
	}
	if !r.Fork || r.ParentFullName != "rust-lang/book" {
		t.Fatalf("fork/parent: fork=%v parent=%q", r.Fork, r.ParentFullName)
	}
}

func TestUpsertAndQueryStars(t *testing.T) {
	s := tempDB(t)

	s.UpsertStar("r", "2026-01-01", 10)
	s.UpsertStar("r", "2026-01-15", 15)
	s.UpsertStar("r", "2026-02-01", 20)
	// Lower value should not overwrite
	s.UpsertStar("r", "2026-02-01", 18)

	rows, err := s.StarsByRepo("r")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	if rows[2].Total != 20 {
		t.Errorf("last star total = %d, want 20 (MAX kept)", rows[2].Total)
	}
}

func TestUpdateDeltas(t *testing.T) {
	s := tempDB(t)

	s.UpsertReferrer("r", "2026-03-20", "google.com", 40, 10)
	s.UpsertReferrer("r", "2026-03-21", "google.com", 50, 15)
	s.UpsertReferrer("r", "2026-03-22", "google.com", 55, 16)

	if err := s.UpdateDeltas(); err != nil {
		t.Fatal(err)
	}

	// Check deltas via PopularReferrers (uses count_delta/uniques_delta)
	items, err := s.PopularReferrers("r", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	// Day 1: delta = MAX(0, 40-0)=40, Day 2: 50-40=10, Day 3: 55-50=5 => total 55
	if items[0].Count != 55 {
		t.Errorf("total count_delta = %d, want 55", items[0].Count)
	}
}

func TestHasRepos(t *testing.T) {
	s := tempDB(t)

	has, err := s.HasRepos()
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("empty DB should have no repos")
	}

	s.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	has, _ = s.HasRepos()
	if !has {
		t.Error("should have repos after insert")
	}
}

func TestDateRange(t *testing.T) {
	s := tempDB(t)

	earliest, latest, err := s.DateRange("r")
	if err != nil {
		t.Fatal(err)
	}
	if earliest != "" || latest != "" {
		t.Errorf("empty db: earliest=%q, latest=%q", earliest, latest)
	}

	s.UpsertView("r", "2026-01-15", 1, 1)
	s.UpsertView("r", "2026-03-20", 1, 1)
	s.UpsertView("r", "2026-02-10", 1, 1)

	earliest, latest, err = s.DateRange("r")
	if err != nil {
		t.Fatal(err)
	}
	if earliest != "2026-01-15" {
		t.Errorf("earliest = %q, want 2026-01-15", earliest)
	}
	if latest != "2026-03-20" {
		t.Errorf("latest = %q, want 2026-03-20", latest)
	}
}

func TestIsolationBetweenRepos(t *testing.T) {
	s := tempDB(t)

	s.UpsertView("repo-a", "2026-03-20", 10, 5)
	s.UpsertView("repo-b", "2026-03-20", 99, 50)

	rows, err := s.ViewsByRange("repo-a", "2026-03-20", "2026-03-20")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Count != 10 {
		t.Errorf("repo-a: got count=%d, want 10", rows[0].Count)
	}
}
