package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hrodrig/gghstats/internal/report"
	"github.com/hrodrig/gghstats/internal/store"
)

func runReport(args []string) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	var gf globalFlags
	var days int
	fs.StringVar(&gf.Repo, "repo", envOr("GGHSTATS_REPO", ""), "owner/repo")
	fs.StringVar(&gf.Token, "token", envOr("GGHSTATS_GITHUB_TOKEN", ""), "GitHub personal access token")
	fs.StringVar(&gf.DB, "db", envOr("GGHSTATS_DB", defaultDBPath()), "SQLite database path")
	fs.IntVar(&days, "days", 14, "number of days to show")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if gf.Repo == "" {
		return fmt.Errorf("--repo or GGHSTATS_REPO is required")
	}
	if gf.Token == "" {
		return fmt.Errorf("--token or GGHSTATS_GITHUB_TOKEN is required")
	}

	db, err := store.Open(gf.DB)
	if err != nil {
		return err
	}
	defer db.Close()

	to := time.Now().UTC().Format("2006-01-02")
	from := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")

	views, err := db.ViewsByRange(gf.Repo, from, to)
	if err != nil {
		return fmt.Errorf("query views: %w", err)
	}

	clones, err := db.ClonesByRange(gf.Repo, from, to)
	if err != nil {
		return fmt.Errorf("query clones: %w", err)
	}

	refs, err := db.ReferrersByRange(gf.Repo, from, to)
	if err != nil {
		return fmt.Errorf("query referrers: %w", err)
	}

	paths, err := db.PathsByRange(gf.Repo, from, to)
	if err != nil {
		return fmt.Errorf("query paths: %w", err)
	}

	report.Terminal(os.Stdout, gf.Repo, views, clones, refs, paths)
	return nil
}
