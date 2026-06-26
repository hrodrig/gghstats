package sync

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/github"
)

type countingRec struct {
	counts map[string]int
}

func (c *countingRec) ObserveSyncError(kind string) {
	c.counts[kind]++
}

func TestRunWorkersProcessesAll(t *testing.T) {
	repos := []github.Repo{
		{FullName: "a/1"},
		{FullName: "a/2"},
		{FullName: "a/3"},
		{FullName: "a/4"},
	}
	var seen atomic.Int32
	runWorkers(context.Background(), repos, workerOptions{
		Workers: 2,
		Work: func(_ context.Context, _ github.Repo) error {
			seen.Add(1)
			return nil
		},
	})
	if got := seen.Load(); got != int32(len(repos)) {
		t.Fatalf("processed %d, want %d", got, len(repos))
	}
}

func TestRunWorkersRecordsErrors(t *testing.T) {
	repos := []github.Repo{
		{FullName: "a/1"},
		{FullName: "a/2"},
		{FullName: "a/3"},
	}
	rec := &countingRec{counts: map[string]int{}}
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
