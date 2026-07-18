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

// RunMilestoneRules evaluates star growth ladders after a successful sync (SPEC §8.3).
// Each threshold fires at most once (debounce once).
func RunMilestoneRules(ctx context.Context, cfg EvalConfig) {
	if cfg.DB == nil || len(cfg.Senders) == 0 || len(cfg.Rules) == 0 {
		return
	}
	now := cfg.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	today := now.Format("2006-01-02")

	for _, rule := range cfg.Rules {
		if !rule.isMilestone() {
			continue
		}
		crossings, err := crossedMilestones(cfg.DB, rule)
		if err != nil {
			slog.Error("alert: evaluate milestone", "error", err, "repo", rule.Repo)
			continue
		}
		for _, th := range crossings {
			key := rule.milestoneIdentityKey(th)
			ok, err := shouldFire(cfg.DB, key, "once", today)
			if err != nil {
				slog.Error("alert: debounce read", "error", err, "key", key)
				continue
			}
			if !ok {
				continue
			}
			p := milestonePayload(rule, th, now, cfg.PublicURL)
			if err := FanOut(ctx, cfg.Senders, p); err != nil {
				slog.Error("alert: deliver milestone", "error", err, "repo", p.Repo, "value", p.Value)
				continue
			}
			if err := markFired(cfg.DB, key, "once", today); err != nil {
				slog.Error("alert: debounce write", "error", err, "key", key)
			}
		}
	}
}

// crossedMilestones returns thresholds already reached by current repos.stars, ascending.
func crossedMilestones(db *store.Store, rule RuleSpec) ([]int, error) {
	sum, err := db.RepoByName(rule.Repo)
	if err != nil {
		return nil, err
	}
	if sum == nil {
		return nil, fmt.Errorf("repo %q not found", rule.Repo)
	}
	out := make([]int, 0, len(rule.Milestones))
	for _, t := range rule.Milestones {
		if sum.Stars >= t {
			out = append(out, t)
		}
	}
	return out, nil
}

func milestonePayload(rule RuleSpec, threshold int, now time.Time, publicURL string) Payload {
	next := ""
	for _, t := range rule.Milestones {
		if t > threshold {
			next = strconv.Itoa(t)
			break
		}
	}
	ruleLine := fmt.Sprintf("crossed %d (final)", threshold)
	if next != "" {
		ruleLine = fmt.Sprintf("crossed %d (next %s)", threshold, next)
	}
	p := Payload{
		Kind:    KindTraffic,
		Version: version.Version,
		When:    now,
		Repo:    rule.Repo,
		Metric:  "stars",
		Window:  "milestone",
		Value:   strconv.Itoa(threshold),
		Rule:    ruleLine,
	}
	if publicURL != "" {
		p.Dash = strings.TrimRight(publicURL, "/") + "/repo/" + rule.Repo
	}
	return p
}
