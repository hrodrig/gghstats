package alert

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/hrodrig/gghstats/internal/store"
	"github.com/hrodrig/gghstats/internal/version"
)

// EvalConfig controls post-sync traffic rule evaluation.
type EvalConfig struct {
	DB        *store.Store
	Rules     []RuleSpec
	Senders   []Sender
	PublicURL string
	Now       time.Time // zero = time.Now().UTC()
}

// RunTrafficRules evaluates traffic rules and fans out matching alerts (SPEC §8.2 / §8.4).
// Skips ops and milestone rules. Best-effort delivery; logs FanOut errors.
func RunTrafficRules(ctx context.Context, cfg EvalConfig) {
	if cfg.DB == nil || len(cfg.Senders) == 0 || len(cfg.Rules) == 0 {
		return
	}
	now := cfg.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	today := now.Format("2006-01-02")

	for _, rule := range cfg.Rules {
		if len(rule.Milestones) > 0 {
			slog.Debug("alert: skipping milestone rule (A2+)", "repo", rule.Repo)
			continue
		}
		if rule.Kind == KindOps {
			continue
		}
		payloads, err := evaluateTrafficRule(cfg.DB, rule, now, today, cfg.PublicURL)
		if err != nil {
			slog.Error("alert: evaluate rule", "error", err, "repo", rule.Repo, "metric", rule.Metric)
			continue
		}
		for _, p := range payloads {
			key := rule.identityKey(firstNonEmpty(p.Repo, p.Scope))
			ok, err := shouldFire(cfg.DB, key, rule.debounceMode(), today)
			if err != nil {
				slog.Error("alert: debounce read", "error", err, "key", key)
				continue
			}
			if !ok {
				continue
			}
			if err := FanOut(ctx, cfg.Senders, p); err != nil {
				slog.Error("alert: deliver", "error", err, "repo", p.Repo, "metric", p.Metric)
				continue
			}
			if err := markFired(cfg.DB, key, rule.debounceMode(), today); err != nil {
				slog.Error("alert: debounce write", "error", err, "key", key)
			}
		}
	}
}

func shouldFire(db *store.Store, key, mode, today string) (bool, error) {
	switch mode {
	case "every_sync":
		return true, nil
	case "once":
		stamp, err := db.AlertDebounceGet(key)
		if err != nil {
			return false, err
		}
		return stamp == "", nil
	default: // once_per_utc_day
		stamp, err := db.AlertDebounceGet(key)
		if err != nil {
			return false, err
		}
		return stamp != today, nil
	}
}

func markFired(db *store.Store, key, mode, today string) error {
	switch mode {
	case "every_sync":
		return nil
	case "once":
		return db.AlertDebounceSet(key, "fired")
	default:
		return db.AlertDebounceSet(key, today)
	}
}

func evaluateTrafficRule(db *store.Store, rule RuleSpec, now time.Time, today, publicURL string) ([]Payload, error) {
	switch rule.Scope {
	case "all_repos":
		p, fire, err := evalOne(db, rule, "", now, today, publicURL)
		if err != nil || !fire {
			return nil, err
		}
		return []Payload{p}, nil
	case "each_repo":
		repos, err := db.ListRepos("name", "asc")
		if err != nil {
			return nil, err
		}
		var out []Payload
		for _, rs := range repos {
			p, fire, err := evalOne(db, rule, rs.Name, now, today, publicURL)
			if err != nil {
				return nil, err
			}
			if fire {
				out = append(out, p)
			}
		}
		return out, nil
	default:
		if rule.Repo == "" {
			return nil, fmt.Errorf("repo required when scope unset")
		}
		p, fire, err := evalOne(db, rule, rule.Repo, now, today, publicURL)
		if err != nil || !fire {
			return nil, err
		}
		return []Payload{p}, nil
	}
}

func evalOne(db *store.Store, rule RuleSpec, repo string, now time.Time, today, publicURL string) (Payload, bool, error) {
	value, detail, err := metricValue(db, rule, repo, now, today)
	if err != nil {
		return Payload{}, false, err
	}
	fire, display := compare(rule.Op, value, rule.Value)
	if !fire {
		return Payload{}, false, nil
	}

	p := Payload{
		Kind:    KindTraffic,
		Version: version.Version,
		When:    now,
		Metric:  rule.Metric,
		Window:  rule.Window,
		Value:   display,
		Rule:    fmt.Sprintf("%s %g", rule.Op, rule.Value),
	}
	if rule.Scope == "all_repos" {
		p.Scope = "all_repos"
	} else {
		p.Repo = repo
		if publicURL != "" {
			p.Dash = strings.TrimRight(publicURL, "/") + "/repo/" + repo
		}
	}
	if detail != "" {
		// Append to rule line for WoW context
		p.Rule = p.Rule + " (" + detail + ")"
	}
	return p, true, nil
}

func metricValue(db *store.Store, rule RuleSpec, repo string, now time.Time, today string) (float64, string, error) {
	metric := rule.Metric
	window := rule.Window

	if rule.Scope == "all_repos" && window == "lifetime" {
		switch metric {
		case "clones":
			n, err := db.SumClonesAll()
			return float64(n), "", err
		case "views":
			n, err := db.SumViewsAll()
			return float64(n), "", err
		default:
			return 0, "", fmt.Errorf("lifetime all_repos unsupported metric %q", metric)
		}
	}

	if repo == "" && rule.Scope != "all_repos" {
		return 0, "", fmt.Errorf("repo required")
	}

	table := metric
	if metric != "clones" && metric != "views" {
		return 0, "", fmt.Errorf("unsupported metric %q (MVP: clones, views)", metric)
	}

	switch window {
	case "1d":
		n, err := db.DayCount(table, repo, today)
		return float64(n), "", err
	case "7d":
		n, err := sumRange(db, table, repo, now, 0, 7)
		return float64(n), "", err
	case "30d":
		n, err := sumRange(db, table, repo, now, 0, 30)
		return float64(n), "", err
	case "lifetime":
		if metric == "clones" {
			sum, err := db.RepoByName(repo)
			if err != nil || sum == nil {
				return 0, "", err
			}
			return float64(sum.TotalClones), "", nil
		}
		sum, err := db.RepoByName(repo)
		if err != nil || sum == nil {
			return 0, "", err
		}
		return float64(sum.TotalViews), "", nil
	case "wow":
		cur, err := sumRange(db, table, repo, now, 0, 7)
		if err != nil {
			return 0, "", err
		}
		prev, err := sumRange(db, table, repo, now, 7, 7)
		if err != nil {
			return 0, "", err
		}
		if prev < 1 {
			prev = 1
		}
		dropPct := (1.0 - float64(cur)/float64(prev)) * 100.0
		detail := fmt.Sprintf("this_week=%d last_week=%d", cur, prev)
		return dropPct, detail, nil
	default:
		return 0, "", fmt.Errorf("unsupported window %q", window)
	}
}

func sumRange(db *store.Store, table, repo string, now time.Time, daysBackEnd, spanDays int) (int, error) {
	end := now.AddDate(0, 0, -daysBackEnd)
	start := end.AddDate(0, 0, -(spanDays - 1))
	from, to := start.Format("2006-01-02"), end.Format("2006-01-02")
	var rows []store.DayRow
	var err error
	if table == "clones" {
		rows, err = db.ClonesByRange(repo, from, to)
	} else {
		rows, err = db.ViewsByRange(repo, from, to)
	}
	if err != nil {
		return 0, err
	}
	sum := 0
	for _, r := range rows {
		sum += r.Count
	}
	return sum, nil
}

func compare(op string, value, threshold float64) (bool, string) {
	switch op {
	case "gte":
		return value >= threshold, formatNum(value)
	case "gt":
		return value > threshold, formatNum(value)
	case "lte":
		return value <= threshold, formatNum(value)
	case "lt":
		return value < threshold, formatNum(value)
	case "eq":
		return value == threshold, formatNum(value)
	case "drop_pct":
		// value is already drop percent; fire when drop >= threshold
		return value >= threshold, fmt.Sprintf("-%.0f%%", math.Round(value))
	default:
		return false, formatNum(value)
	}
}

func formatNum(v float64) string {
	if v == math.Trunc(v) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}
