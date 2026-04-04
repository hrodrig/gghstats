package main

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/hrodrig/gghstats/internal/version"
)

func TestLinePrefixWriter(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	lw := &linePrefixWriter{w: &buf}
	log := slog.New(slog.NewTextHandler(lw, &slog.HandlerOptions{Level: slog.LevelInfo}))
	log.Info("hello", "k", 1)
	out := buf.String()
	if !strings.HasPrefix(out, logLinePrefix) {
		t.Fatalf("want line prefixed with %q, got %q", logLinePrefix, out)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("expected message in output: %q", out)
	}
}

func TestMaskGitHubToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"", "(empty)"},
		{"short", "[masked]"},
		{"ghp_12", "[masked]"},
		{"12345678", "1234....5678"},
		{"ghp_abcdefghijklm", "ghp_....jklm"},
	}
	for _, tt := range tests {
		if got := maskGitHubToken(tt.in); got != tt.want {
			t.Errorf("maskGitHubToken(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestServeLogLevel(t *testing.T) {
	tests := []struct {
		env    string
		want   slog.Level
		isInfo bool // default branch
	}{
		{"debug", slog.LevelDebug, false},
		{"DEBUG", slog.LevelDebug, false},
		{"warn", slog.LevelWarn, false},
		{"warning", slog.LevelWarn, false},
		{"error", slog.LevelError, false},
		{"", slog.LevelInfo, true},
		{"other", slog.LevelInfo, true},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv("GGHSTATS_LOG_LEVEL", tt.env)
			if got := serveLogLevel(); got != tt.want {
				t.Errorf("serveLogLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteServeStartupBanner(t *testing.T) {
	t.Setenv("GGHSTATS_LOG_LEVEL", "")
	oldV, oldD := version.Version, version.BuildDate
	version.Version = "0.9.9"
	version.BuildDate = "2026-01-01"
	t.Cleanup(func() {
		version.Version, version.BuildDate = oldV, oldD
	})

	var buf bytes.Buffer
	writeServeStartupBanner(&buf, serveConfig{
		Host:        "127.0.0.1",
		Port:        "9999",
		GithubToken: "1234567890123456",
	})
	out := buf.String()
	if !strings.Contains(out, "0.9.9") || !strings.Contains(out, "127.0.0.1:9999") {
		t.Errorf("banner: %q", out)
	}
	if !strings.Contains(out, "1234....3456") {
		t.Errorf("expected masked token in %q", out)
	}
}

func TestInitServeLogging(t *testing.T) {
	old := slog.Default()
	t.Cleanup(func() { slog.SetDefault(old) })

	t.Setenv("GGHSTATS_LOG_LEVEL", "warn")
	initServeLogging()
	slog.Warn("test init logging")
	// Global handler is wired; absence of panic is enough for coverage.
}

func TestWriteServeStartupBannerEmptyToken(t *testing.T) {
	oldV, oldD := version.Version, version.BuildDate
	version.Version = "v"
	version.BuildDate = "d"
	t.Cleanup(func() {
		version.Version, version.BuildDate = oldV, oldD
	})

	var buf bytes.Buffer
	writeServeStartupBanner(&buf, serveConfig{Host: "0.0.0.0", Port: "80", GithubToken: ""})
	if !strings.Contains(buf.String(), "(empty)") {
		t.Errorf("want (empty) token: %q", buf.String())
	}
}
