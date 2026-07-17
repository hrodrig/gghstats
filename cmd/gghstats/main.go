package main

import (
	"fmt"
	"os"

	"github.com/hrodrig/gghstats/contrib"
	"github.com/hrodrig/gghstats/internal/version"
)

const usage = `gghstats — GitHub Traffic Stats Collector

Usage:
  gghstats <command> [flags]
  gghstats --print-sample-config

Commands:
  serve    Start web dashboard with auto-sync scheduler
  run      Alias for serve (local convenience)
  fetch    Fetch traffic data from GitHub API and store locally
  report   Display traffic summary in the terminal
  export   Export traffic data to CSV
  backup   Snapshot the SQLite database (VACUUM INTO)
  restore  Replace the SQLite database from a backup file
  version  Print version information

CLI flags (fetch/report/export):
  --repo   owner/repo      Repository (or GGHSTATS_REPO env var)
  --token  TOKEN           GitHub token (or GGHSTATS_GITHUB_TOKEN env var)
  --db     PATH            SQLite database path (default: ./data/gghstats.db)

Backup / restore:
  --db     PATH            SQLite database path (or GGHSTATS_DB)
  --output PATH            backup: destination file (required)
  --input  PATH            restore: backup file to copy from (required)

Server (gghstats serve or gghstats run):
  --port PORT              Listen port (overrides GGHSTATS_PORT; default 8080)
  --open                   Open the default browser when the server is ready
  --demo                   Sample data UI; no GitHub token (or GGHSTATS_DEMO=true)

Server env vars (serve):
  GGHSTATS_GITHUB_TOKEN        GitHub personal access token (required unless demo)
  GGHSTATS_DEMO                true = demo mode (sample data; no token / no sync)
  GGHSTATS_DB                  SQLite path (default: ./data/gghstats.db)
  GGHSTATS_HOST                Bind address (default: 127.0.0.1; use 0.0.0.0 in Docker)
  GGHSTATS_PORT                Listen port (default: 8080)
  GGHSTATS_FILTER              Repo filter (default: * = all)
  GGHSTATS_INCLUDE_PRIVATE     Include private repos (default false; 1/true/yes/on)
  GGHSTATS_SYNC_INTERVAL       Sync frequency (default: 1h)
  GGHSTATS_SYNC_ON_STARTUP     Run full sync before serving (default: true; false = use existing DB, sync later)
  GGHSTATS_SYNC_WORKERS        Concurrent repos per sync cycle (default: 4; same as --sync-workers)
  GGHSTATS_API_TOKEN           Protect /api/repos (and badges when GGHSTATS_BADGE_PUBLIC=false)
  GGHSTATS_BADGE_PUBLIC        Badge SVG public (default true; set false to require x-api-token)
  GGHSTATS_BADGE_CACHE_SECONDS Cache-Control max-age for badge SVG (default: 300)
  GGHSTATS_PUBLIC_URL          Optional public base URL (badges, /robots.txt, /sitemap.xml)
  GGHSTATS_OPEN_BROWSER        Open default browser on startup (default false; same as --open)
  GGHSTATS_LOG_LEVEL           Log level: debug, info (default), warn, error
  GGHSTATS_METRICS             Set to false to disable GET /metrics (Prometheus); default enabled
  GGHSTATS_METRICS_PER_REPO    Set to true to expose per-repo gauges (higher cardinality)
  GGHSTATS_CUSTOM_CSS          Optional .css path: overrides / extends neo-brutalist app.css (simpler UI, branding)
  GGHSTATS_DEFAULT_LOCALE      Dashboard default locale (default: en)
  GGHSTATS_ENABLED_LOCALES     Comma-separated UI locales (default: en,es,de)
  GGHSTATS_HEAD_HTML           Raw HTML injected just before </head> on every page (analytics, extra CSS)
  GGHSTATS_REVERSE_PROXY_RULES JSON array of reverse-proxy rules. See contrib/gghstats.env.example for format.
  GGHSTATS_ENABLE_COLLECTOR    Opt-in anonymous startup telemetry (default false)
  GGHSTATS_ENABLE_UPDATE_CHECK Set false to disable startup newer-release check (default true)
  GGHSTATS_WHITELIST           Comma-separated IPs/CIDRs allowed (empty = all)
  GGHSTATS_WHITELIST_PATHS     Path prefixes where whitelist applies (empty = all routes)
  GGHSTATS_RATE_LIMIT_ENABLED  Per-IP rate limit (default true; set false to disable)
  GGHSTATS_RATE_LIMIT_REQUESTS Requests per window (default: 120)
  GGHSTATS_RATE_LIMIT_PERIOD   Rate-limit window duration (default: 1m)
  GGHSTATS_RATE_LIMIT_BURST    Burst size (default: 20)

  Booleans accept 1/true/yes/on and 0/false/no/off. Full detail: man gghstats, README, contrib/gghstats.env.example.

Run 'gghstats <command> --help' for command-specific flags.`

func main() {
	os.Exit(runCLI(os.Args))
}

type cliCmd func(args []string) error

var cliCommands = map[string]cliCmd{
	"serve":   runServe,
	"run":     runServe,
	"fetch":   runFetch,
	"report":  runReport,
	"export":  runExport,
	"backup":  runBackup,
	"restore": runRestore,
}

// runCLI runs the CLI and returns a process exit code (0 = success).
func runCLI(args []string) int {
	if len(args) >= 2 && isPrintSampleConfigArg(args[1]) {
		fmt.Print(contrib.SampleEnv())
		return 0
	}
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		return 1
	}

	cmd := args[1]
	switch cmd {
	case "version":
		fmt.Printf("gghstats %s (commit: %s, built: %s)\n",
			version.Version, version.Commit, version.BuildDate)
		return 0
	case "--help", "-h", "help":
		fmt.Println(usage)
		return 0
	}

	run, ok := cliCommands[cmd]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n%s\n", cmd, usage)
		return 1
	}
	if err := run(args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func isPrintSampleConfigArg(s string) bool {
	return s == "--print-sample-config" || s == "-print-sample-config"
}
