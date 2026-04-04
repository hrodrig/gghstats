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
