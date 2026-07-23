package main

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/hrodrig/gghstats/internal/server"
)

func TestWarnAPIOnlyOpenCORS(t *testing.T) {
	var buf bytes.Buffer
	old := slog.Default()
	t.Cleanup(func() { slog.SetDefault(old) })
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))

	warnAPIOnlyOpenCORS(false, nil)
	if buf.Len() != 0 {
		t.Fatalf("no warn when not API-only: %s", buf.String())
	}

	buf.Reset()
	warnAPIOnlyOpenCORS(true, []string{"https://app.example"})
	if buf.Len() != 0 {
		t.Fatalf("no warn with closed CORS: %s", buf.String())
	}

	buf.Reset()
	warnAPIOnlyOpenCORS(true, nil)
	if !strings.Contains(buf.String(), "API-only mode with open CORS") {
		t.Fatalf("want open CORS warn, got %s", buf.String())
	}

	buf.Reset()
	warnAPIOnlyOpenCORS(true, []string{"*"})
	if !strings.Contains(buf.String(), "API-only mode with open CORS") {
		t.Fatalf("want * CORS warn, got %s", buf.String())
	}
}

func TestWarnCSPEnforceWithHeadHTML(t *testing.T) {
	var buf bytes.Buffer
	old := slog.Default()
	t.Cleanup(func() { slog.SetDefault(old) })
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))

	warnCSPEnforceWithHeadHTML("enforce", "")
	if buf.Len() != 0 {
		t.Fatalf("no warn without HeadHTML: %s", buf.String())
	}

	buf.Reset()
	warnCSPEnforceWithHeadHTML("report-only", "<script></script>")
	if buf.Len() != 0 {
		t.Fatalf("no warn when not enforce: %s", buf.String())
	}

	buf.Reset()
	warnCSPEnforceWithHeadHTML("enforce", "<script></script>")
	if !strings.Contains(buf.String(), "GGHSTATS_CSP=enforce ignored") {
		t.Fatalf("want CSP warn, got %s", buf.String())
	}
}

func TestLoadServeConfigAPIOnlyCORSAndCSP(t *testing.T) {
	t.Setenv("GGHSTATS_API_ONLY", "true")
	t.Setenv("GGHSTATS_CORS_ORIGINS", "https://a.example, https://b.example")
	t.Setenv("GGHSTATS_CSP", "enforce")

	cfg := loadServeConfig()
	if !cfg.APIOnly {
		t.Fatal("APIOnly want true")
	}
	if cfg.CORSOrigins != "https://a.example, https://b.example" {
		t.Fatalf("CORSOrigins = %q", cfg.CORSOrigins)
	}
	if cfg.CSPMode != "enforce" {
		t.Fatalf("CSPMode = %q", cfg.CSPMode)
	}
	origins := server.ParseCORSOrigins(cfg.CORSOrigins)
	if len(origins) != 2 {
		t.Fatalf("parsed origins = %#v", origins)
	}
}
