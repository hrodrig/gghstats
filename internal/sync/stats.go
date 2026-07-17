package sync

import "sync"

// RunResult summarizes one sync cycle for ops alerts (SPEC §8.7).
type RunResult struct {
	Success            bool
	ReposAttempted     int
	ReposFailed        int
	FailedRepos        []string // capped sample of owner/name
	Unreachable        bool
	RateLimitRemaining int // -1 if unknown
}

// repoFailCounter counts per-repo failures while forwarding ErrRecorder metrics.
type repoFailCounter struct {
	mu      sync.Mutex
	failed  int
	names   []string
	inner   ErrRecorder
	maxName int
}

func newRepoFailCounter(inner ErrRecorder) *repoFailCounter {
	return &repoFailCounter{inner: inner, maxName: 5}
}

func (c *repoFailCounter) ObserveSyncError(kind string) {
	if c.inner != nil {
		c.inner.ObserveSyncError(kind)
	}
}

func (c *repoFailCounter) ObserveSyncRepo(status string) {
	if c.inner != nil {
		c.inner.ObserveSyncRepo(status)
	}
}

func (c *repoFailCounter) noteFail(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failed++
	if len(c.names) < c.maxName {
		c.names = append(c.names, name)
	}
}

func (c *repoFailCounter) snapshot() (int, []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := append([]string(nil), c.names...)
	return c.failed, out
}
