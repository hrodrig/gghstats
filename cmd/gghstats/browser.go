package main

import (
	"log/slog"
	"os/exec"
	"runtime"
)

// serveDashboardURL returns a URL suitable for opening in a local browser.
func serveDashboardURL(host, port string) string {
	switch host {
	case "0.0.0.0", "::", "[::]", "":
		return "http://127.0.0.1:" + port
	default:
		return "http://" + host + ":" + port
	}
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
