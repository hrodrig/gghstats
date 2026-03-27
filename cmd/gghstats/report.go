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
	fs, gf, err := parseGlobalFlags("report", args)
	if err != nil {
		return err
	}

	days := 14
	fs.IntVar(&days, "days", 14, "number of days to show")
	// Re-parse to pick up report-specific flags.
	// parseGlobalFlags already consumed known flags; re-parse remaining.
	_ = fs // days already parsed if provided before global flags

	db, err := store.Open(gf.DB)
	if err != nil {
		return err
	}
	defer db.Close()

	to := time.Now().UTC().Format("2006-01-02")
	from := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")

	// Allow --days to be passed; re-parse remaining args for it
	rfs := flag.NewFlagSet("report-extra", flag.ContinueOnError)
	rfs.IntVar(&days, "days", 14, "")
	_ = rfs.Parse(fs.Args())
	from = time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")

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
