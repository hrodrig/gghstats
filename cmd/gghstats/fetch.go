package main

import (
	"fmt"
	"time"

	"github.com/hrodrig/gghstats/internal/github"
	"github.com/hrodrig/gghstats/internal/store"
)

func runFetch(args []string) error {
	_, gf, err := parseGlobalFlags("fetch", args)
	if err != nil {
		return err
	}

	gh := github.NewClient(gf.Token)

	db, err := store.Open(gf.DB)
	if err != nil {
		return err
	}
	defer db.Close()

	today := time.Now().UTC().Format("2006-01-02")

	if err := upsertRepoFromGitHub(gh, db, gf.Repo); err != nil {
		return err
	}
	if err := fetchStoreViews(gh, db, gf.Repo); err != nil {
		return err
	}
	if err := fetchStoreClones(gh, db, gf.Repo); err != nil {
		return err
	}
	if err := fetchStoreReferrers(gh, db, gf.Repo, today); err != nil {
		return err
	}
	if err := fetchStorePaths(gh, db, gf.Repo, today); err != nil {
		return err
	}
	fmt.Printf("\nData saved to %s\n", gf.DB)
	return nil
}

func upsertRepoFromGitHub(gh *github.Client, db *store.Store, repo string) error {
	meta, err := gh.Repo(repo)
	if err != nil {
		return fmt.Errorf("fetch repo metadata: %w", err)
	}
	prs, err := gh.OpenPullRequests(repo)
	if err != nil {
		prs = nil
	}
	issuesOnly := meta.OpenIssuesCount - len(prs)
	if issuesOnly < 0 {
		issuesOnly = 0
	}
	if err := db.UpsertRepo(
		meta.FullName, meta.DescriptionOrEmpty(),
		meta.StargazersCount, meta.ForksCount, meta.WatchersCount,
		issuesOnly, len(prs),
		meta.Fork, meta.Archived,
		meta.ParentFullName(),
	); err != nil {
		return fmt.Errorf("store repo metadata: %w", err)
	}
	return nil
}

func fetchStoreViews(gh *github.Client, db *store.Store, repo string) error {
	views, err := gh.Views(repo)
	if err != nil {
		return fmt.Errorf("fetch views: %w", err)
	}
	for _, v := range views.Views {
		d := v.Timestamp.Format("2006-01-02")
		if err := db.UpsertView(repo, d, v.Count, v.Uniques); err != nil {
			return fmt.Errorf("store view %s: %w", d, err)
		}
	}
	fmt.Printf("views:     %d days stored (total: %d, uniques: %d)\n",
		len(views.Views), views.Count, views.Uniques)
	return nil
}

func fetchStoreClones(gh *github.Client, db *store.Store, repo string) error {
	clones, err := gh.Clones(repo)
	if err != nil {
		return fmt.Errorf("fetch clones: %w", err)
	}
	for _, c := range clones.Clones {
		d := c.Timestamp.Format("2006-01-02")
		if err := db.UpsertClone(repo, d, c.Count, c.Uniques); err != nil {
			return fmt.Errorf("store clone %s: %w", d, err)
		}
	}
	fmt.Printf("clones:    %d days stored (total: %d, uniques: %d)\n",
		len(clones.Clones), clones.Count, clones.Uniques)
	return nil
}

func fetchStoreReferrers(gh *github.Client, db *store.Store, repo, today string) error {
	refs, err := gh.Referrers(repo)
	if err != nil {
		return fmt.Errorf("fetch referrers: %w", err)
	}
	for _, r := range refs {
		if err := db.UpsertReferrer(repo, today, r.Referrer, r.Count, r.Uniques); err != nil {
			return fmt.Errorf("store referrer %s: %w", r.Referrer, err)
		}
	}
	fmt.Printf("referrers: %d entries stored\n", len(refs))
	return nil
}

func fetchStorePaths(gh *github.Client, db *store.Store, repo, today string) error {
	paths, err := gh.PopularPaths(repo)
	if err != nil {
		return fmt.Errorf("fetch paths: %w", err)
	}
	for _, p := range paths {
		if err := db.UpsertPath(repo, today, p.Path, p.Title, p.Count, p.Uniques); err != nil {
			return fmt.Errorf("store path %s: %w", p.Path, err)
		}
	}
	fmt.Printf("paths:     %d entries stored\n", len(paths))
	return nil
}
