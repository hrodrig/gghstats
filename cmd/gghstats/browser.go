package main

import (
	"log/slog"
	"os/exec"
	"runtime"
)

// serveDashboardURL returns the base HTTP URL for local bind/open helpers.
func serveDashboardURL(host, port string) string {
	switch host {
	case "0.0.0.0", "::", "[::]", "":
		return "http://127.0.0.1:" + port
	default:
		return "http://" + host + ":" + port
	}
}

// serveOpenURL returns the URL opened by --open / GGHSTATS_OPEN_BROWSER.
// API-only mode has no HTML at `/`, so open healthz instead.
func serveOpenURL(host, port string, apiOnly bool) string {
	base := serveDashboardURL(host, port)
	if apiOnly {
		return base + "/api/v1/healthz"
	}
	return base
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		slog.Warn("could not open browser", "url", url, "error", err)
	}
}
