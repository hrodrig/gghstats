package sync

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/hrodrig/gghstats/internal/github"
	"github.com/hrodrig/gghstats/internal/store"
)

// Options configures a sync run.
type Options struct {
	Repos          []string // explicit repos; empty = auto-discover
	IncludePrivate bool
	Filter         string // filter expression (e.g. "hrodrig/*,!fork")
	SyncStars      bool   // fetch stargazer history (expensive for large repos)
	Workers        int    // concurrent repos. 1 = serial, 0 = 1 (sequential). Caller sets default.
}

// Run performs a full sync cycle: discover repos, fetch metrics, store.
// When opts.Workers >= 2, repos are processed concurrently by a worker
// pool. Errors from individual repos are logged and counted but do not
// abort the cycle. The returned RunResult always describes the cycle
// (even when err != nil from resolve failure).
func Run(gh *github.Client, db *store.Store, opts Options, rec ErrRecorder) (RunResult, error) {
	result := RunResult{RateLimitRemaining: -1}
	if gh != nil {
		result.RateLimitRemaining = gh.LastRateLimitRemaining()
	}

	repos, err := resolveRepos(gh, opts)
	if err != nil {
		result.Unreachable = isUnreachableErr(err)
		result.Success = false
		return result, fmt.Errorf("resolve repos: %w", err)
	}

	if len(repos) == 0 {
		slog.Warn("no repos to sync")
		result.Success = true
		return result, nil
	}

	today := time.Now().UTC().Format("2006-01-02")
	slog.Info("sync started", "repos", len(repos), "date", today, "workers", workerCount(opts.Workers))

	counter := newRepoFailCounter(rec)
	runWorkers(context.Background(), repos, workerOptions{
		Workers: workerCount(opts.Workers),
		Metrics: counter,
		Work: func(ctx context.Context, r github.Repo) error {
			return syncRepo(gh, db, r, today, opts.SyncStars, counter)
		},
	})

	result.ReposAttempted = len(repos)
	result.ReposFailed, result.FailedRepos = counter.snapshot()
	if gh != nil {
		result.RateLimitRemaining = gh.LastRateLimitRemaining()
	}

	if err := db.UpdateDeltasSince(today); err != nil {
		slog.Error("update deltas failed", "error", err)
	}

	slog.Info("sync completed", "repos", len(repos), "failed", result.ReposFailed)
	result.Success = true
	return result, nil
}

func isUnreachableErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, needle := range []string{
		"connection refused", "no such host", "i/o timeout", "timeout",
		"network is unreachable", "tls handshake", "dial tcp", "temporary failure",
		"connection reset", "eof",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

// workerCount normalizes the user-supplied worker count to a safe minimum.
// Zero or negative values collapse to 1 (serial); callers should pick a
// sensible default (e.g. 4) when leaving the field unset.
func workerCount(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func resolveRepos(gh *github.Client, opts Options) ([]github.Repo, error) {
	if len(opts.Repos) > 0 {
		// Build minimal Repo structs for explicit list — metadata will be fetched per-repo
		var repos []github.Repo
		for _, name := range opts.Repos {
			repos = append(repos, github.Repo{FullName: name})
		}
		return repos, nil
	}

	all, err := gh.ListRepos(opts.IncludePrivate)
	if err != nil {
		return nil, err
	}

	if opts.Filter == "" || opts.Filter == "*" {
		return all, nil
	}

	return applyFilter(all, opts.Filter), nil
}

func syncRepo(gh *github.Client, db *store.Store, repo github.Repo, today string, syncStars bool, rec ErrRecorder) error {
	name := repo.FullName
	slog.Info("syncing", "repo", name)

	repo, err := ensureRepoMetadata(gh, repo, rec)
	if err != nil {
		return err
	}
	if err := upsertRepoRecord(gh, db, repo, rec); err != nil {
		return err
	}
	syncRepoTraffic(gh, db, name, rec)
	syncRepoSnapshots(gh, db, name, today, rec)
	syncRepoStars(gh, db, repo, name, today, syncStars, rec)
	return nil
}

func ensureRepoMetadata(gh *github.Client, repo github.Repo, rec ErrRecorder) (github.Repo, error) {
	if repo.ID != 0 {
		return repo, nil
	}
	meta, err := gh.Repo(repo.FullName)
	if err != nil {
		recordSyncErr(rec, "repo_meta")
		return repo, fmt.Errorf("repo metadata: %w", err)
	}
	return *meta, nil
}

func upsertRepoRecord(gh *github.Client, db *store.Store, repo github.Repo, rec ErrRecorder) error {
	name := repo.FullName
	prs, err := gh.OpenPullRequests(name)
	if err != nil {
		recordSyncErr(rec, "open_prs")
		slog.Warn("open PRs failed", "repo", name, "error", err)
		prs = nil
	}
	issuesOnly := repo.OpenIssuesCount - len(prs)
	if issuesOnly < 0 {
		issuesOnly = 0
	}
	if err := db.UpsertRepo(
		name, repo.DescriptionOrEmpty(),
		repo.StargazersCount, repo.ForksCount, repo.WatchersCount,
		issuesOnly, len(prs),
		repo.Fork, repo.Archived,
		repo.ParentFullName(),
	); err != nil {
		return fmt.Errorf("upsert repo: %w", err)
	}
	return nil
}

func syncRepoTraffic(gh *github.Client, db *store.Store, name string, rec ErrRecorder) {
	views, err := gh.Views(name)
	if err != nil {
		recordSyncErr(rec, "views")
		slog.Warn("views failed", "repo", name, "error", err)
	} else {
		for _, v := range views.Views {
			d := v.Timestamp.UTC().Format("2006-01-02")
			db.UpsertView(name, d, v.Count, v.Uniques)
		}
	}
	clones, err := gh.Clones(name)
	if err != nil {
		recordSyncErr(rec, "clones")
		slog.Warn("clones failed", "repo", name, "error", err)
	} else {
		for _, c := range clones.Clones {
			d := c.Timestamp.UTC().Format("2006-01-02")
			db.UpsertClone(name, d, c.Count, c.Uniques)
		}
	}
}

func syncRepoSnapshots(gh *github.Client, db *store.Store, name, today string, rec ErrRecorder) {
	refs, err := gh.Referrers(name)
	if err != nil {
		recordSyncErr(rec, "referrers")
		slog.Warn("referrers failed", "repo", name, "error", err)
	} else {
		for _, r := range refs {
			db.UpsertReferrer(name, today, r.Referrer, r.Count, r.Uniques)
		}
	}
	paths, err := gh.PopularPaths(name)
	if err != nil {
		recordSyncErr(rec, "paths")
		slog.Warn("paths failed", "repo", name, "error", err)
	} else {
		for _, p := range paths {
			db.UpsertPath(name, today, p.Path, p.Title, p.Count, p.Uniques)
		}
	}
}

func syncRepoStars(gh *github.Client, db *store.Store, repo github.Repo, name, today string, syncStars bool, rec ErrRecorder) {
	if !syncStars {
		db.UpsertStar(name, today, repo.StargazersCount)
		return
	}

	cursor, err := db.GetStarSyncCursor(name)
	if err != nil {
		recordSyncErr(rec, "stargazers")
		slog.Warn("stargazers cursor read failed", "repo", name, "error", err)
		return
	}

	current := repo.StargazersCount
	if cursor.Synced && current == cursor.LastSeenStarCount {
		slog.Info("stargazers skipped", "repo", name, "reason", "count_unchanged", "count", current)
		return
	}

	// Count dropped (unstars) or first sync: full pagination rebuild.
	if !cursor.Synced || current < cursor.LastSeenStarCount {
		stars, err := gh.Stargazers(name)
		if err != nil {
			recordSyncErr(rec, "stargazers")
			slog.Warn("stargazers failed", "repo", name, "error", err)
			return
		}
		storeStarHistory(db, name, stars)
		if err := db.SetStarSyncCursor(name, current, newestStarredAt(stars)); err != nil {
			slog.Warn("stargazers cursor write failed", "repo", name, "error", err)
		}
		slog.Info("stargazers synced", "repo", name, "mode", "full", "stars", len(stars), "count", current)
		return
	}

	delta := current - cursor.LastSeenStarCount
	stars, err := gh.StargazersRecent(name, delta, cursor.LastStarredAt)
	if err != nil {
		recordSyncErr(rec, "stargazers")
		slog.Warn("stargazers failed", "repo", name, "error", err)
		return
	}
	if len(stars) == 0 && delta > 0 {
		slog.Warn("stargazers incremental empty", "repo", name, "delta", delta)
		return
	}
	storeStarHistoryIncremental(db, name, stars, cursor.LastSeenStarCount)
	newest := newestStarredAt(stars)
	if newest.IsZero() {
		newest = cursor.LastStarredAt
	}
	if err := db.SetStarSyncCursor(name, current, newest); err != nil {
		slog.Warn("stargazers cursor write failed", "repo", name, "error", err)
	}
	slog.Info("stargazers synced", "repo", name, "mode", "incremental", "new", len(stars), "count", current)
}

// recordSyncErr is a nil-safe wrapper that increments the per-kind error
// counter when rec is configured.
func recordSyncErr(rec ErrRecorder, kind string) {
	if rec == nil {
		return
	}
	rec.ObserveSyncError(kind)
}

// storeStarHistory converts individual star events into daily cumulative counts.
// GitHub returns newest-first; we sort ascending so totals grow with time.
func storeStarHistory(db *store.Store, repo string, stars []github.Star) {
	if len(stars) == 0 {
		return
	}
	sorted := append([]github.Star(nil), stars...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StarredAt.Before(sorted[j].StarredAt)
	})
	cumulative := 0
	for _, s := range sorted {
		cumulative++
		date := s.StarredAt.UTC().Format("2006-01-02")
		db.UpsertStar(repo, date, cumulative)
	}
}

// storeStarHistoryIncremental appends newest stars onto an existing cumulative base.
func storeStarHistoryIncremental(db *store.Store, repo string, newStars []github.Star, baseCount int) {
	if len(newStars) == 0 {
		return
	}
	sorted := append([]github.Star(nil), newStars...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StarredAt.Before(sorted[j].StarredAt)
	})
	for i, s := range sorted {
		date := s.StarredAt.UTC().Format("2006-01-02")
		db.UpsertStar(repo, date, baseCount+i+1)
	}
}

func newestStarredAt(stars []github.Star) time.Time {
	var newest time.Time
	for _, s := range stars {
		if s.StarredAt.After(newest) {
			newest = s.StarredAt
		}
	}
	return newest
}

// --- Filter logic ---
// Supports: "owner/*", "owner/repo", "*", "!fork", "!archived", negation with "!"

func applyFilter(repos []github.Repo, filter string) []github.Repo {
	includes, excludes, excludeFork, excludeArchived := parseFilterRules(filter)
	hasDirectIncludes := hasNonWildcardInclude(includes)
	var result []github.Repo
	for _, repo := range repos {
		if shouldIncludeRepo(repo, includes, excludes, excludeFork, excludeArchived, hasDirectIncludes) {
			result = append(result, repo)
		}
	}
	return result
}

func parseFilterRules(filter string) (includes, excludes []string, excludeFork, excludeArchived bool) {
	for _, raw := range strings.Split(filter, ",") {
		rule := strings.TrimSpace(raw)
		if rule == "" {
			continue
		}
		switch {
		case rule == "!fork":
			excludeFork = true
		case rule == "!archived":
			excludeArchived = true
		case strings.HasPrefix(rule, "!"):
			excludes = append(excludes, rule[1:])
		default:
			includes = append(includes, rule)
		}
	}
	return includes, excludes, excludeFork, excludeArchived
}

func hasNonWildcardInclude(includes []string) bool {
	for _, inc := range includes {
		if inc != "*" {
			return true
		}
	}
	return false
}

func shouldIncludeRepo(repo github.Repo, includes, excludes []string, excludeFork, excludeArchived, hasDirectIncludes bool) bool {
	if excludeFork && repo.Fork {
		return false
	}
	if excludeArchived && repo.Archived {
		return false
	}
	if isExcluded(repo.FullName, excludes) {
		return false
	}
	return !hasDirectIncludes || matchesAny(repo.FullName, includes)
}

func matchesAny(name string, patterns []string) bool {
	for _, p := range patterns {
		if p == "*" {
			return true
		}
		if strings.HasSuffix(p, "/*") {
			prefix := p[:len(p)-2]
			if strings.HasPrefix(name, prefix+"/") {
				return true
			}
		} else if p == name {
			return true
		}
	}
	return false
}

func isExcluded(name string, excludes []string) bool {
	for _, ex := range excludes {
		if strings.HasSuffix(ex, "/*") {
			prefix := ex[:len(ex)-2]
			if strings.HasPrefix(name, prefix+"/") {
				return true
			}
		} else if ex == name {
			return true
		}
	}
	return false
}
