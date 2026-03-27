package main

import (
	"flag"
	"fmt"
	"os"
)

type globalFlags struct {
	Repo  string
	Token string
	DB    string
}

func parseGlobalFlags(name string, args []string) (*flag.FlagSet, globalFlags, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	var gf globalFlags

	fs.StringVar(&gf.Repo, "repo", envOr("GGHSTATS_REPO", ""), "owner/repo")
	fs.StringVar(&gf.Token, "token", envOr("GGHSTATS_GITHUB_TOKEN", ""), "GitHub personal access token")
	fs.StringVar(&gf.DB, "db", envOr("GGHSTATS_DB", defaultDBPath()), "SQLite database path")

	if err := fs.Parse(args); err != nil {
		return fs, gf, err
	}

	if gf.Repo == "" {
		return fs, gf, fmt.Errorf("--repo or GGHSTATS_REPO is required")
	}
	if gf.Token == "" {
		return fs, gf, fmt.Errorf("--token or GGHSTATS_GITHUB_TOKEN is required")
	}

	return fs, gf, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func defaultDBPath() string {
	return "./data/gghstats.db"
}
