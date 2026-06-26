package sync

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/github"
)

type countingRec struct {
	mu     sync.Mutex
	counts map[string]int
	repos  map[string]int
}

func (c *countingRec) ObserveSyncError(kind string) {
	c.mu.Lock()
	c.counts[kind]++
	c.mu.Unlock()
}

func (c *countingRec) ObserveSyncRepo(status string) {
	c.mu.Lock()
	c.repos[status]++
	c.mu.Unlock()
}

func TestRunWorkersProcessesAll(t *testing.T) {
	repos := []github.Repo{
		{FullName: "a/1"},
		{FullName: "a/2"},
		{FullName: "a/3"},
		{FullName: "a/4"},
	}
	var seen atomic.Int32
	rec := &countingRec{counts: map[string]int{}, repos: map[string]int{}}
	runWorkers(context.Background(), repos, workerOptions{
		Workers: 2,
		Metrics: rec,
		Work: func(_ context.Context, _ github.Repo) error {
			seen.Add(1)
			return nil
		},
	})
	if got := seen.Load(); got != int32(len(repos)) {
		t.Fatalf("processed %d, want %d", got, len(repos))
	}
	if rec.repos["success"] != len(repos) {
		t.Fatalf("success count = %d, want %d (repos=%v)", rec.repos["success"], len(repos), rec.repos)
	}
	if rec.repos["error"] != 0 {
		t.Fatalf("error count = %d, want 0", rec.repos["error"])
	}
}

func TestRunWorkersRecordsErrors(t *testing.T) {
	repos := []github.Repo{
		{FullName: "a/1"},
		{FullName: "a/2"},
		{FullName: "a/3"},
	}
	rec := &countingRec{counts: map[string]int{}, repos: map[string]int{}}
	runWorkers(context.Background(), repos, workerOptions{
		Workers: 2,
		Metrics: rec,
		Work: func(_ context.Context, r github.Repo) error {
			if r.FullName == "a/2" {
				return errSentinel
			}
			return nil
		},
	})
	if rec.counts["worker"] != 1 {
		t.Fatalf("errors = %d, want 1", rec.counts["worker"])
	}
	if rec.repos["error"] != 1 {
		t.Fatalf("repo error count = %d, want 1", rec.repos["error"])
	}
	if rec.repos["success"] != 2 {
		t.Fatalf("repo success count = %d, want 2", rec.repos["success"])
	}
}

func TestRunWorkersNilRecNoPanic(t *testing.T) {
	repos := []github.Repo{{FullName: "a/1"}}
	runWorkers(context.Background(), repos, workerOptions{
		Workers: 1,
		Work: func(_ context.Context, _ github.Repo) error {
			return errSentinel
		},
	})
}

func TestRunWorkersConcurrent(t *testing.T) {
	repos := make([]github.Repo, 16)
	for i := range repos {
		repos[i] = github.Repo{FullName: "a/" + string(rune('a'+i))}
	}
	var peak atomic.Int32
	var cur atomic.Int32
	runWorkers(context.Background(), repos, workerOptions{
		Workers: 4,
		Work: func(_ context.Context, _ github.Repo) error {
			n := cur.Add(1)
			for {
				old := peak.Load()
				if n <= old || peak.CompareAndSwap(old, n) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			cur.Add(-1)
			return nil
		},
	})
	if got := peak.Load(); got < 2 {
		t.Fatalf("peak concurrency = %d, want >= 2", got)
	}
	if got := peak.Load(); got > 4 {
		t.Fatalf("peak concurrency = %d, want <= 4", got)
	}
}

func TestWorkerCountDefaults(t *testing.T) {
	if got := workerCount(0); got != 1 {
		t.Fatalf("workerCount(0) = %d, want 1", got)
	}
	if got := workerCount(-1); got != 1 {
		t.Fatalf("workerCount(-1) = %d, want 1", got)
	}
	if got := workerCount(8); got != 8 {
		t.Fatalf("workerCount(8) = %d, want 8", got)
	}
}

var errSentinel = sentinelErr("boom")

type sentinelErr string

func (e sentinelErr) Error() string { return string(e) }
