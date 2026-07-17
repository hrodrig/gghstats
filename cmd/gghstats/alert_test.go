package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/alert"
)

func TestRunCLI_alertTest_noSinks(t *testing.T) {
	t.Setenv("GGHSTATS_ALERT_SINKS", "")
	t.Setenv("GGHSTATS_ALERTS_ENABLED", "")
	if code := runCLI([]string{"gghstats", "alert", "test"}); code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestRunCLI_alertTest_deliveryOK(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv("GGHSTATS_ALERT_SINKS", `[{"type":"slack","webhook_url":"`+srv.URL+`"}]`)
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	code := runCLI([]string{"gghstats", "alert", "test", "--kind", "traffic"})
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	if code != 0 {
		t.Fatalf("want 0, got %d out=%s", code, out)
	}
	if n != 1 {
		t.Fatalf("posts=%d", n)
	}
	if !strings.Contains(string(out), `alert test: sent kind "traffic"`) {
		t.Fatalf("stdout=%q", out)
	}
}

func TestRunCLI_alertTest_deliveryFailExit4(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	alert.ApplyRetryConfig(alert.RetryConfig{MaxAttempts: 1, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond})
	t.Cleanup(func() { alert.ApplyRetryConfig(alert.DefaultRetryConfig) })

	t.Setenv("GGHSTATS_ALERT_SINKS", `[{"type":"slack","webhook_url":"`+srv.URL+`"}]`)
	if code := runCLI([]string{"gghstats", "alert", "test"}); code != 4 {
		t.Fatalf("want exit 4, got %d", code)
	}
}
