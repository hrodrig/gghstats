package main

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
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
