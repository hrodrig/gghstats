package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRunCLIUsageAndErrors(t *testing.T) {
	t.Parallel()
	if code := runCLI([]string{"gghstats"}); code != 1 {
		t.Fatalf("no subcommand: want exit 1, got %d", code)
	}
	if code := runCLI([]string{"gghstats", "nope"}); code != 1 {
		t.Fatalf("unknown command: want exit 1, got %d", code)
	}
}

// TestRunCLISubcommandFailures exercises runCLI dispatch for each subcommand on
// validation errors (no t.Parallel: env is per-test).
func TestRunCLISubcommandFailures(t *testing.T) {
	t.Run("fetch_missing_repo", func(t *testing.T) {
		t.Setenv("GGHSTATS_REPO", "")
		t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
		if code := runCLI([]string{"gghstats", "fetch"}); code != 1 {
			t.Fatalf("want exit 1, got %d", code)
		}
	})
	t.Run("fetch_missing_token", func(t *testing.T) {
		t.Setenv("GGHSTATS_REPO", "owner/name")
		t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
		if code := runCLI([]string{"gghstats", "fetch"}); code != 1 {
			t.Fatalf("want exit 1, got %d", code)
		}
	})
	t.Run("export_missing_repo", func(t *testing.T) {
		t.Setenv("GGHSTATS_REPO", "")
		t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
		if code := runCLI([]string{"gghstats", "export"}); code != 1 {
			t.Fatalf("want exit 1, got %d", code)
		}
	})
	t.Run("report_missing_repo", func(t *testing.T) {
		t.Setenv("GGHSTATS_REPO", "")
		t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
		if code := runCLI([]string{"gghstats", "report"}); code != 1 {
			t.Fatalf("want exit 1, got %d", code)
		}
	})
	t.Run("serve_missing_token", func(t *testing.T) {
		t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
		if code := runCLI([]string{"gghstats", "serve"}); code != 1 {
			t.Fatalf("want exit 1, got %d", code)
		}
	})
}

func TestRunCLIHelp(t *testing.T) {
	stdoutSwapMu.Lock()
	defer stdoutSwapMu.Unlock()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	errCh := make(chan error, 1)
	var out bytes.Buffer
	go func() {
		_, e := io.Copy(&out, r)
		errCh <- e
	}()

	code := runCLI([]string{"gghstats", "help"})
	w.Close()
	os.Stdout = old
	if e := <-errCh; e != nil {
		t.Fatal(e)
	}
	if code != 0 {
		t.Fatalf("help: want 0, got %d", code)
	}
	if !strings.Contains(out.String(), "gghstats") {
		t.Fatalf("expected usage text, got %q", out.String())
	}
}

func TestRunCLIVersion(t *testing.T) {
	stdoutSwapMu.Lock()
	defer stdoutSwapMu.Unlock()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	errCh := make(chan error, 1)
	var out bytes.Buffer
	go func() {
		_, e := io.Copy(&out, r)
		errCh <- e
	}()

	code := runCLI([]string{"gghstats", "version"})
	w.Close()
	os.Stdout = old
	if e := <-errCh; e != nil {
		t.Fatal(e)
	}
	if code != 0 {
		t.Fatalf("version: want 0, got %d", code)
	}
	if !strings.Contains(out.String(), "gghstats") {
		t.Fatalf("expected version line, got %q", out.String())
	}
}
