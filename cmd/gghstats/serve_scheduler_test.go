package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	stdsync "sync"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/github"
	"github.com/hrodrig/gghstats/internal/store"
	"github.com/hrodrig/gghstats/internal/sync"
)

func TestStartSchedulerStopsOnCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/repos" {
			json.NewEncoder(w).Encode([]github.Repo{})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	s, err := store.Open(filepath.Join(t.TempDir(), "sched.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	c := github.NewClient("tok")
	c.BaseURL = srv.URL

	ctx, cancel := context.WithCancel(context.Background())
	var wg stdsync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startScheduler(ctx, c, s, sync.Options{Filter: "*"}, 24*time.Hour)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("scheduler did not stop after cancel")
	}
}

func TestStartSchedulerRunsScheduledSync(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/repos" {
			json.NewEncoder(w).Encode([]github.Repo{})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	s, err := store.Open(filepath.Join(t.TempDir(), "sched-tick.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	c := github.NewClient("tok")
	c.BaseURL = srv.URL

	ctx, cancel := context.WithCancel(context.Background())
	var wg stdsync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startScheduler(ctx, c, s, sync.Options{Filter: "*"}, 25*time.Millisecond)
	}()

	time.Sleep(90 * time.Millisecond)
	cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("scheduler did not stop after cancel")
	}
}
