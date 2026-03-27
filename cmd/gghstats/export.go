package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hrodrig/gghstats/internal/report"
	"github.com/hrodrig/gghstats/internal/store"
)

func runExport(args []string) error {
	fs, gf, err := parseGlobalFlags("export", args)
	if err != nil {
		return err
	}

	days := 14
	output := ""

	rfs := flag.NewFlagSet("export-extra", flag.ContinueOnError)
	rfs.IntVar(&days, "days", 14, "number of days to export")
	rfs.StringVar(&output, "output", "", "output file path (default: stdout)")
	_ = rfs.Parse(fs.Args())

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

	w := os.Stdout
	if output != "" {
		f, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		w = f
	}

	if err := report.CSV(w, gf.Repo, views, clones, refs, paths); err != nil {
		return fmt.Errorf("write CSV: %w", err)
	}

	if output != "" {
		fmt.Fprintf(os.Stderr, "Exported to %s\n", output)
	}
	return nil
}
