package main

import (
	"fmt"
	"os"

	"github.com/hrodrig/gghstats/internal/version"
)

const usage = `gghstats — GitHub Traffic Stats Collector

Usage:
  gghstats <command> [flags]

Commands:
  serve    Start web dashboard with auto-sync scheduler
  fetch    Fetch traffic data from GitHub API and store locally
  report   Display traffic summary in the terminal
  export   Export traffic data to CSV
  version  Print version information

CLI flags (fetch/report/export):
  --repo   owner/repo      Repository (or GGHSTATS_REPO env var)
  --token  TOKEN           GitHub token (or GGHSTATS_GITHUB_TOKEN env var)
  --db     PATH            SQLite database path (default: ./data/gghstats.db)

Server (gghstats serve):
  --port PORT              Listen port (overrides GGHSTATS_PORT; default 8080)

Server env vars (serve):
  GGHSTATS_GITHUB_TOKEN    GitHub personal access token (required)
  GGHSTATS_DB              SQLite path (default: ./data/gghstats.db)
  GGHSTATS_HOST            Bind address (default: 0.0.0.0)
  GGHSTATS_PORT            Listen port (default: 8080)
  GGHSTATS_FILTER          Repo filter (default: * = all)
  GGHSTATS_SYNC_INTERVAL   Sync frequency (default: 1h)
  GGHSTATS_API_TOKEN       Protect /api/* endpoints
  GGHSTATS_LOG_LEVEL       Log level: debug, info (default), warn, error
  GGHSTATS_METRICS         Set to false to disable GET /metrics (Prometheus); default enabled

Run 'gghstats <command> --help' for command-specific flags.`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		if err := runServe(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "fetch":
		if err := runFetch(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "report":
		if err := runReport(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "export":
		if err := runExport(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Printf("gghstats %s (commit: %s, built: %s)\n",
			version.Version, version.Commit, version.BuildDate)
	case "--help", "-h", "help":
		fmt.Println(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n%s\n", os.Args[1], usage)
		os.Exit(1)
	}
}
