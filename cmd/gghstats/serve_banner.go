package main

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/hrodrig/gghstats/internal/version"
)

// logLinePrefix is written at the start of every slog line (structured logs, stderr).
const logLinePrefix = "gghstats "

// linePrefixWriter buffers stderr output and prefixes each full line with logLinePrefix.
type linePrefixWriter struct {
	w   io.Writer
	acc []byte
}

func (p *linePrefixWriter) Write(b []byte) (int, error) {
	n := len(b)
	p.acc = append(p.acc, b...)
	for {
		idx := bytes.IndexByte(p.acc, '\n')
		if idx < 0 {
			return n, nil
		}
		line := p.acc[:idx+1]
		p.acc = append([]byte{}, p.acc[idx+1:]...)
		if _, err := p.w.Write(append([]byte(logLinePrefix), line...)); err != nil {
			return n, err
		}
	}
}

// maskGitHubToken returns the first 4 and last 4 runes of the token separated by "....".
// Shorter tokens are not partially revealed.
func maskGitHubToken(tok string) string {
	if tok == "" {
		return "(empty)"
	}
	r := []rune(tok)
	if len(r) < 8 {
		return "[masked]"
	}
	const keep = 4
	return string(r[:keep]) + "...." + string(r[len(r)-keep:])
}

func writeServeStartupBanner(w io.Writer, cfg serveConfig) {
	addr := cfg.Host + ":" + cfg.Port
	fmt.Fprintf(w, "gghstats %s | build %s | platform %s/%s | listen %s | github_token %s\n",
		version.Version,
		version.BuildDate,
		runtime.GOOS, runtime.GOARCH,
		addr,
		maskGitHubToken(cfg.GithubToken),
	)
}

func serveLogLevel() slog.Level {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("GGHSTATS_LOG_LEVEL"))) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func initServeLogging() {
	lw := &linePrefixWriter{w: os.Stderr}
	h := slog.NewTextHandler(lw, &slog.HandlerOptions{
		Level: serveLogLevel(),
	})
	slog.SetDefault(slog.New(h))
}
