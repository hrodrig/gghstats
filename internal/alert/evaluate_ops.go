package alert

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/hrodrig/gghstats/internal/store"
	"github.com/hrodrig/gghstats/internal/version"
)

const consecutiveKey = "ops|sync_failed|consecutive_count"

// SyncSnapshot is the post-sync view used by ops rules (SPEC §8.7).
type SyncSnapshot struct {
	Success            bool
	ReposAttempted     int
	ReposFailed        int
	FailedRepos        []string
	Unreachable        bool
	RateLimitRemaining int
}

// RunOpsRules evaluates kind=ops rules against a sync snapshot (SPEC §8.7).
func RunOpsRules(ctx context.Context, cfg EvalConfig, snap SyncSnapshot) {
	if cfg.DB == nil || len(cfg.Senders) == 0 || len(cfg.Rules) == 0 {
		return
	}
	now := cfg.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	today := now.Format("2006-01-02")

	consec, err := updateConsecutiveFailures(cfg.DB, snap.Success)
	if err != nil {
		slog.Error("alert: consecutive sync counter", "error", err)
	}

	for _, rule := range cfg.Rules {
		if rule.Kind != KindOps {
			continue
		}
		p, fire, err := evaluateOpsRule(rule, snap, consec, now)
		if err != nil {
			slog.Error("alert: evaluate ops rule", "error", err, "event", rule.Event)
			continue
		}
		if !fire {
			continue
		}
		mode := rule.opsDebounceMode()
		key := rule.identityKey(rule.Event)
		ok, err := shouldFire(cfg.DB, key, mode, today)
		if err != nil {
			slog.Error("alert: debounce read", "error", err, "key", key)
			continue
		}
		if !ok {
			continue
		}
		if err := FanOut(ctx, cfg.Senders, p); err != nil {
			slog.Error("alert: deliver ops", "error", err, "event", rule.Event)
			continue
		}
		if err := markFired(cfg.DB, key, mode, today); err != nil {
			slog.Error("alert: debounce write", "error", err, "key", key)
		}
	}
}

func (r RuleSpec) opsDebounceMode() string {
	if r.Debounce != "" {
		return r.debounceMode()
	}
	if r.Event == "sync_failed" && r.Level == "crit" {
		return "every_sync"
	}
	if r.Event == "github_unreachable" && (r.Level == "crit" || r.Level == "") {
		return "every_sync"
	}
	return "once_per_utc_day"
}

func updateConsecutiveFailures(db *store.Store, success bool) (int, error) {
	if success {
		return 0, db.AlertDebounceSet(consecutiveKey, "0")
	}
	stamp, err := db.AlertDebounceGet(consecutiveKey)
	if err != nil {
		return 0, err
	}
	n := 0
	if stamp != "" {
		n, _ = strconv.Atoi(stamp)
	}
	n++
	if err := db.AlertDebounceSet(consecutiveKey, strconv.Itoa(n)); err != nil {
		return n, err
	}
	return n, nil
}

func evaluateOpsRule(rule RuleSpec, snap SyncSnapshot, consecutive int, now time.Time) (Payload, bool, error) {
	level := rule.Level
	if level == "" {
		level = defaultOpsLevel(rule.Event)
	}
	var count float64
	var detail string
	window := rule.Window
	if window == "" {
		window = "this_sync"
	}

	switch rule.Event {
	case "repo_fetch_failed":
		count = float64(snap.ReposFailed)
		detail = fmt.Sprintf("%d/%d repos failed", snap.ReposFailed, snap.ReposAttempted)
		if len(snap.FailedRepos) > 0 {
			detail += " sample=[" + strings.Join(snap.FailedRepos, ", ") + "]"
		}
		if window != "this_sync" {
			return Payload{}, false, fmt.Errorf("repo_fetch_failed window must be this_sync")
		}
	case "sync_failed":
		if window == "consecutive_runs" {
			count = float64(consecutive)
			detail = fmt.Sprintf("consecutive_failed_runs=%d", consecutive)
		} else {
			if snap.Success {
				return Payload{}, false, nil
			}
			count = 1
			detail = "this sync failed"
			window = "this_sync"
		}
	case "github_unreachable":
		if !snap.Unreachable {
			return Payload{}, false, nil
		}
		count = 1
		detail = "github unreachable (resolve/network)"
		window = "this_sync"
	case "rate_limit":
		if snap.RateLimitRemaining < 0 {
			return Payload{}, false, nil
		}
		count = float64(snap.RateLimitRemaining)
		detail = fmt.Sprintf("remaining=%d", snap.RateLimitRemaining)
	default:
		return Payload{}, false, fmt.Errorf("unknown ops event %q", rule.Event)
	}

	fire, display := compare(rule.Op, count, rule.Value)
	if !fire {
		return Payload{}, false, nil
	}

	p := Payload{
		Kind:      KindOps,
		Version:   version.Version,
		When:      now,
		Level:     level,
		Event:     rule.Event,
		Count:     display,
		Threshold: fmt.Sprintf("%s %g", rule.Op, rule.Value),
		Window:    window,
		Detail:    detail,
	}
	return p, true, nil
}

func defaultOpsLevel(event string) string {
	switch event {
	case "sync_failed", "github_unreachable":
		return "crit"
	default:
		return "warn"
	}
}

// RunAllRules runs ops rules always and traffic rules when the sync succeeded.
func RunAllRules(ctx context.Context, cfg EvalConfig, snap SyncSnapshot) {
	RunOpsRules(ctx, cfg, snap)
	if snap.Success {
		RunTrafficRules(ctx, cfg)
	}
}
