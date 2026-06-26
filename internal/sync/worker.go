package sync

import (
	"context"
	"log/slog"
	"sync"

	"github.com/hrodrig/gghstats/internal/github"
)

// ErrRecorder is the optional hook WorkerPool uses to count sync errors by kind.
// It is satisfied by *metrics.Domain. Pass nil to disable error metrics.
type ErrRecorder interface {
	ObserveSyncError(kind string)
	ObserveSyncRepo(status string)
}

// workerOptions configures runWorkers behavior.
type workerOptions struct {
	Workers int         // number of concurrent repo workers
	Metrics ErrRecorder // optional; nil disables error counting
	Work    func(context.Context, github.Repo) error
}

// runWorkers processes repos with up to opts.Workers goroutines.
//
// It splits work across the pool, returning when every repo has either
// completed or failed. Errors are logged but do not abort the pool — each
// worker keeps going so a single bad repo cannot stall the cycle.
//
// When opts.Metrics is non-nil, every error from opts.Work is recorded via
// ObserveSyncError("worker") so transient and persistent failures both
// surface in Prometheus.
func runWorkers(ctx context.Context, repos []github.Repo, opts workerOptions) {
	if opts.Workers < 1 {
		opts.Workers = 1
	}
	if opts.Work == nil {
		return
	}

	jobs := make(chan github.Repo)
	var wg sync.WaitGroup
	wg.Add(opts.Workers)

	for i := 0; i < opts.Workers; i++ {
		go func() {
			defer wg.Done()
			for repo := range jobs {
				if err := ctx.Err(); err != nil {
					return
				}
				if err := opts.Work(ctx, repo); err != nil {
					if opts.Metrics != nil {
						opts.Metrics.ObserveSyncError("worker")
						opts.Metrics.ObserveSyncRepo("error")
					}
					slog.Error("sync repo failed", "repo", repo.FullName, "error", err)
				} else if opts.Metrics != nil {
					opts.Metrics.ObserveSyncRepo("success")
				}
			}
		}()
	}

	for _, repo := range repos {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return
		case jobs <- repo:
		}
	}
	close(jobs)
	wg.Wait()
}
