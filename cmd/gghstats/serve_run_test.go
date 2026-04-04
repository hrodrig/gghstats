package main

import (
	"testing"
)

func TestRunServeMissingToken(t *testing.T) {
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
	if err := runServe(nil); err == nil {
		t.Fatal("expected error when GGHSTATS_GITHUB_TOKEN is empty")
	}
}

func TestRunServeHelp(t *testing.T) {
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
	if err := runServe([]string{"-h"}); err != nil {
		t.Fatalf("help: %v", err)
	}
	if err := runServe([]string{"--help"}); err != nil {
		t.Fatalf("--help: %v", err)
	}
}
