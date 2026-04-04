package main

import (
	"os"
	"testing"
)

func TestDefaultDBPath(t *testing.T) {
	t.Parallel()
	if got := defaultDBPath(); got != "./data/gghstats.db" {
		t.Errorf("defaultDBPath() = %q", got)
	}
}

func TestEnvOr(t *testing.T) {
	key := "GGHSTATS_TEST_ENV_OR_" + t.Name()
	t.Setenv(key, "")
	if got := envOr(key, "fallback"); got != "fallback" {
		t.Errorf("unset: got %q, want fallback", got)
	}
	t.Setenv(key, "fromenv")
	if got := envOr(key, "fallback"); got != "fromenv" {
		t.Errorf("set: got %q, want fromenv", got)
	}
}

func TestParseGlobalFlagsOK(t *testing.T) {
	repoKey := "GGHSTATS_REPO"
	tokenKey := "GGHSTATS_GITHUB_TOKEN"
	dbKey := "GGHSTATS_DB"
	t.Setenv(repoKey, "")
	t.Setenv(tokenKey, "")
	t.Setenv(dbKey, "")

	_, gf, err := parseGlobalFlags("test", []string{"-repo", "o/r", "-token", "tok", "-db", "/tmp/x.db"})
	if err != nil {
		t.Fatalf("parseGlobalFlags: %v", err)
	}
	if gf.Repo != "o/r" || gf.Token != "tok" || gf.DB != "/tmp/x.db" {
		t.Fatalf("gf = %+v", gf)
	}
}

func TestParseGlobalFlagsFromEnv(t *testing.T) {
	repoKey := "GGHSTATS_REPO"
	tokenKey := "GGHSTATS_GITHUB_TOKEN"
	dbKey := "GGHSTATS_DB"
	t.Cleanup(func() {
		_ = os.Unsetenv(repoKey)
		_ = os.Unsetenv(tokenKey)
		_ = os.Unsetenv(dbKey)
	})
	t.Setenv(repoKey, "env/o")
	t.Setenv(tokenKey, "env-tok")
	t.Setenv(dbKey, "/env.db")

	_, gf, err := parseGlobalFlags("test", []string{})
	if err != nil {
		t.Fatalf("parseGlobalFlags: %v", err)
	}
	if gf.Repo != "env/o" || gf.Token != "env-tok" || gf.DB != "/env.db" {
		t.Fatalf("gf = %+v", gf)
	}
}

func TestParseGlobalFlagsMissingRepo(t *testing.T) {
	t.Setenv("GGHSTATS_REPO", "")
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
	t.Setenv("GGHSTATS_DB", "")

	_, _, err := parseGlobalFlags("test", []string{"-token", "x"})
	if err == nil {
		t.Fatal("expected error when repo missing")
	}
}

func TestParseGlobalFlagsMissingToken(t *testing.T) {
	t.Setenv("GGHSTATS_REPO", "")
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
	t.Setenv("GGHSTATS_DB", "")

	_, _, err := parseGlobalFlags("test", []string{"-repo", "o/r"})
	if err == nil {
		t.Fatal("expected error when token missing")
	}
}

func TestParseGlobalFlagsInvalidFlag(t *testing.T) {
	t.Setenv("GGHSTATS_REPO", "")
	t.Setenv("GGHSTATS_GITHUB_TOKEN", "")
	t.Setenv("GGHSTATS_DB", "")

	_, _, err := parseGlobalFlags("test", []string{"-not-a-real-flag"})
	if err == nil {
		t.Fatal("expected parse error for unknown flag")
	}
}
