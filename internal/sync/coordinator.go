package sync

import (
	"errors"
	"sync"
	"time"

	"github.com/hrodrig/gghstats/internal/github"
	"github.com/hrodrig/gghstats/internal/metrics"
	"github.com/hrodrig/gghstats/internal/store"
)

// ErrInProgress is returned when a sync is already running.
var ErrInProgress = errors.New("sync already in progress")

// Status is a snapshot of manual/scheduled sync activity.
type Status struct {
	Running        bool       `json:"running"`
	Scope          string     `json:"scope,omitempty"` // "all" or "repo"
	Repo           string     `json:"repo,omitempty"`  // set when scope is "repo"
	LastStartedAt  *time.Time `json:"last_started_at,omitempty"`
	LastFinishedAt *time.Time `json:"last_finished_at,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
}

// Coordinator runs sync cycles with at-most-one execution at a time.
type Coordinator struct {
	mu  sync.Mutex
	gh  *github.Client
	db  *store.Store
	opt Options

	metrics   *metrics.Domain
	afterSync func(success bool)

	running        bool
	scope          string
	repo           string
	syncStartedAt  time.Time
	lastStartedAt  time.Time
	lastFinishedAt time.Time
	lastError      string
}

// SetMetrics attaches optional Prometheus domain metrics.
func (c *Coordinator) SetMetrics(m *metrics.Domain) {
	c.metrics = m
}

// SetAfterSync registers a callback invoked after each sync finishes (success or failure).
// Used for post-sync alert evaluation. Must not block forever.
func (c *Coordinator) SetAfterSync(fn func(success bool)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.afterSync = fn
}

// NewCoordinator wires a GitHub client and store for serialized sync runs.
func NewCoordinator(gh *github.Client, db *store.Store, opt Options) *Coordinator {
	return &Coordinator{gh: gh, db: db, opt: opt}
}

// Status returns the current sync state.
func (c *Coordinator) Status() Status {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snapshotLocked()
}

func (c *Coordinator) snapshotLocked() Status {
	st := Status{
		Running:   c.running,
		Scope:     c.scope,
		Repo:      c.repo,
		LastError: c.lastError,
	}
	if !c.lastStartedAt.IsZero() {
		t := c.lastStartedAt.UTC()
		st.LastStartedAt = &t
	}
	if !c.lastFinishedAt.IsZero() {
		t := c.lastFinishedAt.UTC()
		st.LastFinishedAt = &t
	}
	return st
}

// Run performs a full sync cycle, blocking until it finishes. Returns ErrInProgress if one is already running.
func (c *Coordinator) Run() error {
	return c.runBlocking(c.opt, "all", "")
}

// Start launches a full background sync. Returns ErrInProgress if one is already running.
func (c *Coordinator) Start() error {
	return c.startBackground(c.opt, "all", "")
}

// StartRepo syncs a single owner/repo in the background.
func (c *Coordinator) StartRepo(fullName string) error {
	opt := c.opt
	opt.Repos = []string{fullName}
	return c.startBackground(opt, "repo", fullName)
}

func (c *Coordinator) runBlocking(opt Options, scope, repo string) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return ErrInProgress
	}
	c.markRunningLocked(scope, repo)
	c.mu.Unlock()

	err := Run(c.gh, c.db, opt, c.metrics)
	c.finishRun(err)
	return err
}

func (c *Coordinator) startBackground(opt Options, scope, repo string) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return ErrInProgress
	}
	c.markRunningLocked(scope, repo)
	c.mu.Unlock()

	go func() {
		err := Run(c.gh, c.db, opt, c.metrics)
		c.finishRun(err)
	}()
	return nil
}

func (c *Coordinator) markRunningLocked(scope, repo string) {
	c.running = true
	c.scope = scope
	c.repo = repo
	now := time.Now().UTC()
	c.syncStartedAt = now
	c.lastStartedAt = now
	c.lastError = ""
}

func (c *Coordinator) finishRun(err error) {
	c.mu.Lock()
	started := c.syncStartedAt
	dom := c.metrics
	after := c.afterSync
	c.running = false
	c.scope = ""
	c.repo = ""
	c.lastFinishedAt = time.Now().UTC()
	if err != nil {
		c.lastError = err.Error()
	}
	c.mu.Unlock()

	if dom != nil && !started.IsZero() {
		dom.ObserveSync(time.Since(started), err == nil)
	}
	if after != nil {
		after(err == nil)
	}
}
