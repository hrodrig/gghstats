package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// Store wraps a SQLite database for traffic data persistence.
type Store struct {
	db *sql.DB
}

// Open creates or opens the SQLite database at path, creating parent dirs
// if needed, and runs schema migrations.
func Open(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying *sql.DB (for tests or advanced queries).
func (s *Store) DB() *sql.DB {
	return s.db
}

// --- Versioned migrations using PRAGMA user_version ---

type migrationFunc func(db *sql.DB) error

func (s *Store) migrate() error {
	migrations := []migrationFunc{
		migrateV1,
		migrateV2,
		migrateV3,
	}

	var current int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&current); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}

	for i, mig := range migrations {
		ver := i + 1
		if current >= ver {
			continue
		}
		if err := mig(s.db); err != nil {
			return fmt.Errorf("migration v%d: %w", ver, err)
		}
		if _, err := s.db.Exec(fmt.Sprintf("PRAGMA user_version = %d", ver)); err != nil {
			return fmt.Errorf("set user_version %d: %w", ver, err)
		}
	}
	return nil
}

func migrateV1(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS views (
			repo    TEXT NOT NULL,
			date    TEXT NOT NULL,
			count   INTEGER NOT NULL,
			uniques INTEGER NOT NULL,
			PRIMARY KEY (repo, date)
		)`,
		`CREATE TABLE IF NOT EXISTS clones (
			repo    TEXT NOT NULL,
			date    TEXT NOT NULL,
			count   INTEGER NOT NULL,
			uniques INTEGER NOT NULL,
			PRIMARY KEY (repo, date)
		)`,
		`CREATE TABLE IF NOT EXISTS referrers (
			repo     TEXT NOT NULL,
			date     TEXT NOT NULL,
			referrer TEXT NOT NULL,
			count    INTEGER NOT NULL,
			uniques  INTEGER NOT NULL,
			PRIMARY KEY (repo, date, referrer)
		)`,
		`CREATE TABLE IF NOT EXISTS paths (
			repo    TEXT NOT NULL,
			date    TEXT NOT NULL,
			path    TEXT NOT NULL,
			title   TEXT NOT NULL,
			count   INTEGER NOT NULL,
			uniques INTEGER NOT NULL,
			PRIMARY KEY (repo, date, path)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func migrateV2(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS repos (
			name        TEXT PRIMARY KEY,
			description TEXT DEFAULT '',
			stars       INTEGER NOT NULL DEFAULT 0,
			forks       INTEGER NOT NULL DEFAULT 0,
			watchers    INTEGER NOT NULL DEFAULT 0,
			issues      INTEGER NOT NULL DEFAULT 0,
			prs         INTEGER NOT NULL DEFAULT 0,
			fork        INTEGER NOT NULL DEFAULT 0,
			archived    INTEGER NOT NULL DEFAULT 0,
			hidden      INTEGER NOT NULL DEFAULT 0,
			updated_at  TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS stars (
			repo  TEXT NOT NULL,
			date  TEXT NOT NULL,
			total INTEGER NOT NULL,
			PRIMARY KEY (repo, date)
		)`,
		`ALTER TABLE referrers ADD COLUMN count_delta INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE referrers ADD COLUMN uniques_delta INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE paths ADD COLUMN count_delta INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE paths ADD COLUMN uniques_delta INTEGER NOT NULL DEFAULT 0`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func migrateV3(db *sql.DB) error {
	_, err := db.Exec(`ALTER TABLE repos ADD COLUMN parent_full_name TEXT NOT NULL DEFAULT ''`)
	return err
}

// --- Upsert methods ---

// UpsertView inserts or replaces a single day of view data.
func (s *Store) UpsertView(repo, date string, count, uniques int) error {
	_, err := s.db.Exec(
		`INSERT INTO views (repo, date, count, uniques)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT (repo, date) DO UPDATE SET
		   count=MAX(views.count, excluded.count),
		   uniques=MAX(views.uniques, excluded.uniques)`,
		repo, date, count, uniques,
	)
	return err
}

// UpsertClone inserts or replaces a single day of clone data.
func (s *Store) UpsertClone(repo, date string, count, uniques int) error {
	_, err := s.db.Exec(
		`INSERT INTO clones (repo, date, count, uniques)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT (repo, date) DO UPDATE SET
		   count=MAX(clones.count, excluded.count),
		   uniques=MAX(clones.uniques, excluded.uniques)`,
		repo, date, count, uniques,
	)
	return err
}

// UpsertReferrer inserts or replaces a referrer entry for a given date.
func (s *Store) UpsertReferrer(repo, date, referrer string, count, uniques int) error {
	_, err := s.db.Exec(
		`INSERT INTO referrers (repo, date, referrer, count, uniques)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (repo, date, referrer) DO UPDATE SET
		   count=MAX(referrers.count, excluded.count),
		   uniques=MAX(referrers.uniques, excluded.uniques)`,
		repo, date, referrer, count, uniques,
	)
	return err
}

// UpsertPath inserts or replaces a popular path entry for a given date.
func (s *Store) UpsertPath(repo, date, path, title string, count, uniques int) error {
	_, err := s.db.Exec(
		`INSERT INTO paths (repo, date, path, title, count, uniques)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT (repo, date, path) DO UPDATE SET
		   title=excluded.title,
		   count=MAX(paths.count, excluded.count),
		   uniques=MAX(paths.uniques, excluded.uniques)`,
		repo, date, path, title, count, uniques,
	)
	return err
}

// UpsertRepo inserts or updates repo metadata.
// parentFullName is the immediate upstream (e.g. "rust-lang/book"); empty if not a fork.
func (s *Store) UpsertRepo(name, description string, stars, forks, watchers, issues, prs int, fork, archived bool, parentFullName string) error {
	if !fork {
		parentFullName = ""
	}
	_, err := s.db.Exec(
		`INSERT INTO repos (name, description, stars, forks, watchers, issues, prs, fork, archived, parent_full_name, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		 ON CONFLICT (name) DO UPDATE SET
		   description=excluded.description,
		   stars=MAX(repos.stars, excluded.stars),
		   forks=MAX(repos.forks, excluded.forks),
		   watchers=MAX(repos.watchers, excluded.watchers),
		   issues=excluded.issues,
		   prs=excluded.prs,
		   fork=excluded.fork,
		   archived=excluded.archived,
		   parent_full_name=excluded.parent_full_name,
		   hidden=0,
		   updated_at=excluded.updated_at`,
		name, description, stars, forks, watchers, issues, prs, boolToInt(fork), boolToInt(archived), parentFullName,
	)
	return err
}

// UpsertStar inserts or updates the cumulative star count for a repo on a date.
func (s *Store) UpsertStar(repo, date string, total int) error {
	_, err := s.db.Exec(
		`INSERT INTO stars (repo, date, total)
		 VALUES (?, ?, ?)
		 ON CONFLICT (repo, date) DO UPDATE SET total=MAX(stars.total, excluded.total)`,
		repo, date, total,
	)
	return err
}

// UpdateDeltas recalculates count_delta and uniques_delta for referrers and paths
// using the LAG window function.
func (s *Store) UpdateDeltas() error {
	tables := []struct {
		table, col string
	}{
		{"referrers", "referrer"},
		{"paths", "path"},
	}
	for _, t := range tables {
		query := fmt.Sprintf(`
			WITH cte AS (
				SELECT repo, date, %[2]s, uniques, count,
					LAG(uniques) OVER (PARTITION BY repo, %[2]s ORDER BY date) AS prev_uniques,
					LAG(count) OVER (PARTITION BY repo, %[2]s ORDER BY date) AS prev_count
				FROM %[1]s
			)
			UPDATE %[1]s SET
				uniques_delta = MAX(0, cte.uniques - COALESCE(cte.prev_uniques, 0)),
				count_delta = MAX(0, cte.count - COALESCE(cte.prev_count, 0))
			FROM cte
			WHERE %[1]s.repo = cte.repo AND %[1]s.date = cte.date AND %[1]s.%[2]s = cte.%[2]s`,
			t.table, t.col)
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("update deltas for %s: %w", t.table, err)
		}
	}
	return nil
}

// --- Query methods ---

// DayRow represents a single day's aggregated count.
type DayRow struct {
	Date    string `json:"date"`
	Count   int    `json:"count"`
	Uniques int    `json:"uniques"`
}

// ViewsByRange returns view rows for a repo between two dates (inclusive).
func (s *Store) ViewsByRange(repo, from, to string) ([]DayRow, error) {
	return s.queryDayRows(
		`SELECT date, count, uniques FROM views WHERE repo=? AND date>=? AND date<=? ORDER BY date`,
		repo, from, to,
	)
}

// ClonesByRange returns clone rows for a repo between two dates (inclusive).
func (s *Store) ClonesByRange(repo, from, to string) ([]DayRow, error) {
	return s.queryDayRows(
		`SELECT date, count, uniques FROM clones WHERE repo=? AND date>=? AND date<=? ORDER BY date`,
		repo, from, to,
	)
}

// CloneDateExtentForRepos returns the min and max clone dates (YYYY-MM-DD) across the given repos.
// ok is false when repos is empty or no clone rows exist for those repos.
func (s *Store) CloneDateExtentForRepos(repos []string) (minDate, maxDate string, ok bool, err error) {
	if len(repos) == 0 {
		return "", "", false, nil
	}
	ph := placeholders(len(repos))
	args := make([]interface{}, len(repos))
	for i, n := range repos {
		args[i] = n
	}
	q := fmt.Sprintf(`SELECT MIN(date), MAX(date) FROM clones WHERE repo IN (%s)`, ph)
	var minNS, maxNS sql.NullString
	if err := s.db.QueryRow(q, args...).Scan(&minNS, &maxNS); err != nil {
		return "", "", false, err
	}
	if !minNS.Valid || !maxNS.Valid || minNS.String == "" || maxNS.String == "" {
		return "", "", false, nil
	}
	return minNS.String, maxNS.String, true, nil
}

// AggregatedClonesByDayForRepos returns per-day sums of count and uniques across repos (inclusive dates).
// repos must be non-empty; callers should use CloneDateExtentForRepos first when choosing the window.
func (s *Store) AggregatedClonesByDayForRepos(repos []string, from, to string) ([]DayRow, error) {
	if len(repos) == 0 {
		return nil, nil
	}
	ph := placeholders(len(repos))
	q := fmt.Sprintf(
		`SELECT date, SUM(count), SUM(uniques) FROM clones WHERE repo IN (%s) AND date >= ? AND date <= ? GROUP BY date ORDER BY date`,
		ph,
	)
	args := make([]interface{}, 0, len(repos)+2)
	for _, n := range repos {
		args = append(args, n)
	}
	args = append(args, from, to)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DayRow
	for rows.Next() {
		var r DayRow
		if err := rows.Scan(&r.Date, &r.Count, &r.Uniques); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	b := strings.Builder{}
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('?')
	}
	return b.String()
}

// ReferrerRow holds a referrer entry with its date.
type ReferrerRow struct {
	Date     string `json:"date"`
	Referrer string `json:"referrer"`
	Count    int    `json:"count"`
	Uniques  int    `json:"uniques"`
}

// ReferrersByRange returns referrer rows for a repo between two dates.
func (s *Store) ReferrersByRange(repo, from, to string) ([]ReferrerRow, error) {
	rows, err := s.db.Query(
		`SELECT date, referrer, count, uniques FROM referrers
		 WHERE repo=? AND date>=? AND date<=? ORDER BY date, count DESC`,
		repo, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ReferrerRow
	for rows.Next() {
		var r ReferrerRow
		if err := rows.Scan(&r.Date, &r.Referrer, &r.Count, &r.Uniques); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// PathRow holds a popular path entry with its date.
type PathRow struct {
	Date    string `json:"date"`
	Path    string `json:"path"`
	Title   string `json:"title"`
	Count   int    `json:"count"`
	Uniques int    `json:"uniques"`
}

// PathsByRange returns path rows for a repo between two dates.
func (s *Store) PathsByRange(repo, from, to string) ([]PathRow, error) {
	rows, err := s.db.Query(
		`SELECT date, path, title, count, uniques FROM paths
		 WHERE repo=? AND date>=? AND date<=? ORDER BY date, count DESC`,
		repo, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []PathRow
	for rows.Next() {
		var r PathRow
		if err := rows.Scan(&r.Date, &r.Path, &r.Title, &r.Count, &r.Uniques); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// StarRow holds a star count for a date.
type StarRow struct {
	Date  string `json:"date"`
	Total int    `json:"total"`
}

// StarsByRepo returns the star history for a repo.
func (s *Store) StarsByRepo(repo string) ([]StarRow, error) {
	rows, err := s.db.Query(
		`SELECT date, total FROM stars WHERE repo=? ORDER BY date`, repo,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []StarRow
	for rows.Next() {
		var r StarRow
		if err := rows.Scan(&r.Date, &r.Total); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// RepoSummary holds aggregated metrics for a single repo.
type RepoSummary struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Stars          int    `json:"stars"`
	Forks          int    `json:"forks"`
	Watchers       int    `json:"watchers"`
	Issues         int    `json:"issues"`
	PRs            int    `json:"prs"`
	Fork           bool   `json:"fork"`
	ParentFullName string `json:"parent_full_name,omitempty"`
	Archived       bool   `json:"archived"`
	TotalViews     int    `json:"total_views"`
	TotalUniques   int    `json:"total_uniques"`
	TotalClones    int    `json:"total_clones"`
	CloneUniques   int    `json:"clone_uniques"`
}

// Fixed SQL fragments for ListRepos: no dynamic string building from sort/direction reaches the DB (CodeQL-safe).
const listReposSQLBase = `
		SELECT
			r.name, r.description, r.stars, r.forks, r.watchers, r.issues, r.prs,
			r.fork, r.parent_full_name, r.archived,
			COALESCE(v.total_views, 0), COALESCE(v.total_uniques, 0),
			COALESCE(c.total_clones, 0), COALESCE(c.clone_uniques, 0)
		FROM repos r
		LEFT JOIN (
			SELECT repo, SUM(count) AS total_views, SUM(uniques) AS total_uniques
			FROM views GROUP BY repo
		) v ON v.repo = r.name
		LEFT JOIN (
			SELECT repo, SUM(count) AS total_clones, SUM(uniques) AS clone_uniques
			FROM clones GROUP BY repo
		) c ON c.repo = r.name
		WHERE r.hidden = 0
`

const (
	listReposOrderNameAsc         = `ORDER BY r.name ASC`
	listReposOrderNameDesc        = `ORDER BY r.name DESC`
	listReposOrderStarsAsc        = `ORDER BY r.stars ASC`
	listReposOrderStarsDesc       = `ORDER BY r.stars DESC`
	listReposOrderForksAsc        = `ORDER BY r.forks ASC`
	listReposOrderForksDesc       = `ORDER BY r.forks DESC`
	listReposOrderTotalViewsAsc   = `ORDER BY COALESCE(v.total_views, 0) ASC`
	listReposOrderTotalViewsDesc  = `ORDER BY COALESCE(v.total_views, 0) DESC`
	listReposOrderTotalClonesAsc  = `ORDER BY COALESCE(c.total_clones, 0) ASC`
	listReposOrderTotalClonesDesc = `ORDER BY COALESCE(c.total_clones, 0) DESC`
)

// Each queryListRepos* method passes only compile-time constant SQL to db.Query (no query string
// built from request parameters). The dispatcher below selects a method by whitelisted sort keys.
func (s *Store) queryListReposNameAsc() (*sql.Rows, error) {
	return s.db.Query(listReposSQLBase + listReposOrderNameAsc)
}
func (s *Store) queryListReposNameDesc() (*sql.Rows, error) {
	return s.db.Query(listReposSQLBase + listReposOrderNameDesc)
}
func (s *Store) queryListReposStarsAsc() (*sql.Rows, error) {
	return s.db.Query(listReposSQLBase + listReposOrderStarsAsc)
}
func (s *Store) queryListReposStarsDesc() (*sql.Rows, error) {
	return s.db.Query(listReposSQLBase + listReposOrderStarsDesc)
}
func (s *Store) queryListReposForksAsc() (*sql.Rows, error) {
	return s.db.Query(listReposSQLBase + listReposOrderForksAsc)
}
func (s *Store) queryListReposForksDesc() (*sql.Rows, error) {
	return s.db.Query(listReposSQLBase + listReposOrderForksDesc)
}
func (s *Store) queryListReposTotalViewsAsc() (*sql.Rows, error) {
	return s.db.Query(listReposSQLBase + listReposOrderTotalViewsAsc)
}
func (s *Store) queryListReposTotalViewsDesc() (*sql.Rows, error) {
	return s.db.Query(listReposSQLBase + listReposOrderTotalViewsDesc)
}
func (s *Store) queryListReposTotalClonesAsc() (*sql.Rows, error) {
	return s.db.Query(listReposSQLBase + listReposOrderTotalClonesAsc)
}
func (s *Store) queryListReposTotalClonesDesc() (*sql.Rows, error) {
	return s.db.Query(listReposSQLBase + listReposOrderTotalClonesDesc)
}

func (s *Store) queryListReposRows(sort, direction string) (*sql.Rows, error) {
	asc := direction == "asc"
	switch sort {
	case "name":
		if asc {
			return s.queryListReposNameAsc()
		}
		return s.queryListReposNameDesc()
	case "stars":
		if asc {
			return s.queryListReposStarsAsc()
		}
		return s.queryListReposStarsDesc()
	case "forks":
		if asc {
			return s.queryListReposForksAsc()
		}
		return s.queryListReposForksDesc()
	case "total_views":
		if asc {
			return s.queryListReposTotalViewsAsc()
		}
		return s.queryListReposTotalViewsDesc()
	case "total_clones":
		if asc {
			return s.queryListReposTotalClonesAsc()
		}
		return s.queryListReposTotalClonesDesc()
	default:
		return s.queryListReposTotalViewsDesc()
	}
}

// ListRepos returns all non-hidden repos with their aggregated traffic totals.
func (s *Store) ListRepos(sort, direction string) ([]RepoSummary, error) {
	allowed := map[string]bool{
		"name": true, "stars": true, "forks": true,
		"total_views": true, "total_clones": true,
	}
	if !allowed[sort] {
		sort = "total_views"
	}
	if direction != "asc" {
		direction = "desc"
	}

	rows, err := s.queryListReposRows(sort, direction)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []RepoSummary
	for rows.Next() {
		var r RepoSummary
		if err := rows.Scan(
			&r.Name, &r.Description, &r.Stars, &r.Forks, &r.Watchers,
			&r.Issues, &r.PRs, &r.Fork, &r.ParentFullName, &r.Archived,
			&r.TotalViews, &r.TotalUniques, &r.TotalClones, &r.CloneUniques,
		); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// RepoByName returns the summary for a single repo, or nil if not found.
func (s *Store) RepoByName(name string) (*RepoSummary, error) {
	var r RepoSummary
	err := s.db.QueryRow(`
		SELECT
			r.name, r.description, r.stars, r.forks, r.watchers, r.issues, r.prs,
			r.fork, r.parent_full_name, r.archived,
			COALESCE(v.total_views, 0), COALESCE(v.total_uniques, 0),
			COALESCE(c.total_clones, 0), COALESCE(c.clone_uniques, 0)
		FROM repos r
		LEFT JOIN (
			SELECT repo, SUM(count) AS total_views, SUM(uniques) AS total_uniques
			FROM views GROUP BY repo
		) v ON v.repo = r.name
		LEFT JOIN (
			SELECT repo, SUM(count) AS total_clones, SUM(uniques) AS clone_uniques
			FROM clones GROUP BY repo
		) c ON c.repo = r.name
		WHERE r.name = ? AND r.hidden = 0`,
		name,
	).Scan(
		&r.Name, &r.Description, &r.Stars, &r.Forks, &r.Watchers,
		&r.Issues, &r.PRs, &r.Fork, &r.ParentFullName, &r.Archived,
		&r.TotalViews, &r.TotalUniques, &r.TotalClones, &r.CloneUniques,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// HasRepos returns true if there is at least one non-hidden repo in the database.
func (s *Store) HasRepos() (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM repos WHERE hidden = 0").Scan(&count)
	return count > 0, err
}

// DateRange returns the earliest and latest dates in the views table for a repo.
func (s *Store) DateRange(repo string) (earliest, latest string, err error) {
	err = s.db.QueryRow(
		`SELECT COALESCE(MIN(date),''), COALESCE(MAX(date),'') FROM views WHERE repo=?`, repo,
	).Scan(&earliest, &latest)
	return
}

// PopularItem holds an aggregated referrer or path entry.
type PopularItem struct {
	Name    string `json:"name"`
	Count   int64  `json:"count"`
	Uniques int64  `json:"uniques"`
}

// PopularReferrers returns aggregated referrers for a repo within a day range.
func (s *Store) PopularReferrers(repo string, days int) ([]PopularItem, error) {
	return s.queryPopular("referrers", "referrer", repo, days)
}

// PopularPaths returns aggregated paths for a repo within a day range.
func (s *Store) PopularPaths(repo string, days int) ([]PopularItem, error) {
	return s.queryPopular("paths", "path", repo, days)
}

func (s *Store) queryPopular(table, col, repo string, days int) ([]PopularItem, error) {
	dateFilter := "1=1"
	if days > 0 {
		dateFilter = fmt.Sprintf("date >= date('now', '-%d day')", days)
	}
	query := fmt.Sprintf(
		`SELECT %[1]s, SUM(count_delta), SUM(uniques_delta) FROM %[2]s
		 WHERE repo=? AND %[3]s
		 GROUP BY %[1]s
		 ORDER BY SUM(uniques_delta) DESC`,
		col, table, dateFilter)

	rows, err := s.db.Query(query, repo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []PopularItem
	for rows.Next() {
		var r PopularItem
		if err := rows.Scan(&r.Name, &r.Count, &r.Uniques); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (s *Store) queryDayRows(query, repo, from, to string) ([]DayRow, error) {
	rows, err := s.db.Query(query, repo, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DayRow
	for rows.Next() {
		var r DayRow
		if err := rows.Scan(&r.Date, &r.Count, &r.Uniques); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
