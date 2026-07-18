package alert

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// RuleSpec is one entry in GGHSTATS_ALERT_RULES (SPEC §8.4).
type RuleSpec struct {
	Kind       string  `json:"kind,omitempty"` // traffic (default) | ops
	Repo       string  `json:"repo,omitempty"`
	Scope      string  `json:"scope,omitempty"` // all_repos | each_repo
	Metric     string  `json:"metric"`
	Window     string  `json:"window"`
	Op         string  `json:"op"`
	Value      float64 `json:"value"`
	Debounce   string  `json:"debounce,omitempty"`
	Fire       string  `json:"fire,omitempty"`
	Level      string  `json:"level,omitempty"`      // ops
	Event      string  `json:"event,omitempty"`      // ops
	Milestones []int   `json:"milestones,omitempty"` // growth ladder (SPEC §8.3)
}

// ParseRulesJSON parses GGHSTATS_ALERT_RULES.
func ParseRulesJSON(raw string) ([]RuleSpec, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var rules []RuleSpec
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return nil, fmt.Errorf("GGHSTATS_ALERT_RULES: %w", err)
	}
	for i := range rules {
		if err := normalizeRule(&rules[i]); err != nil {
			return nil, fmt.Errorf("GGHSTATS_ALERT_RULES[%d]: %w", i, err)
		}
	}
	return rules, nil
}

func normalizeRule(r *RuleSpec) error {
	r.Kind = strings.ToLower(strings.TrimSpace(r.Kind))
	if r.Kind == "" {
		r.Kind = KindTraffic
	}
	r.Metric = strings.ToLower(strings.TrimSpace(r.Metric))
	r.Window = strings.ToLower(strings.TrimSpace(r.Window))
	r.Op = strings.ToLower(strings.TrimSpace(r.Op))
	r.Scope = strings.ToLower(strings.TrimSpace(r.Scope))
	r.Debounce = strings.ToLower(strings.TrimSpace(r.Debounce))
	r.Fire = strings.ToLower(strings.TrimSpace(r.Fire))
	r.Repo = strings.TrimSpace(r.Repo)
	r.Event = strings.ToLower(strings.TrimSpace(r.Event))
	r.Level = strings.ToLower(strings.TrimSpace(r.Level))

	if len(r.Milestones) > 0 {
		return normalizeMilestoneRule(r)
	}
	switch r.Kind {
	case KindTraffic:
		if r.Metric == "" {
			return fmt.Errorf("metric is required")
		}
		if r.Window == "" {
			return fmt.Errorf("window is required")
		}
		if r.Op == "" {
			return fmt.Errorf("op is required")
		}
		if r.Repo == "" && r.Scope == "" {
			return fmt.Errorf("repo or scope is required")
		}
	case KindOps:
		if r.Event == "" {
			return fmt.Errorf("ops rule requires event")
		}
	default:
		return fmt.Errorf("unknown kind %q", r.Kind)
	}
	return nil
}

func normalizeMilestoneRule(r *RuleSpec) error {
	if r.Kind != KindTraffic && r.Kind != "" {
		return fmt.Errorf("milestone rules must be kind traffic (got %q)", r.Kind)
	}
	r.Kind = KindTraffic
	if r.Metric == "" {
		r.Metric = "stars"
	}
	if r.Metric != "stars" {
		return fmt.Errorf("milestones only support metric=stars (got %q)", r.Metric)
	}
	if r.Repo == "" {
		return fmt.Errorf("milestone rule requires repo")
	}
	cleaned := make([]int, 0, len(r.Milestones))
	seen := make(map[int]struct{}, len(r.Milestones))
	for _, t := range r.Milestones {
		if t <= 0 {
			return fmt.Errorf("milestone thresholds must be positive (got %d)", t)
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		cleaned = append(cleaned, t)
	}
	if len(cleaned) == 0 {
		return fmt.Errorf("milestones must list at least one positive threshold")
	}
	sort.Ints(cleaned)
	r.Milestones = cleaned
	// Fire-once per threshold (SPEC §8.3); ignore once_per_utc_day.
	r.Fire = "once"
	r.Debounce = "once"
	r.Window = "milestone"
	return nil
}

// RulesFromEnv loads GGHSTATS_ALERT_RULES.
func RulesFromEnv(getenv func(string) string) ([]RuleSpec, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	return ParseRulesJSON(getenv("GGHSTATS_ALERT_RULES"))
}

func (r RuleSpec) isMilestone() bool {
	return len(r.Milestones) > 0
}

func (r RuleSpec) debounceMode() string {
	if r.isMilestone() {
		return "once"
	}
	if r.Fire == "once" || r.Debounce == "once" {
		return "once"
	}
	if r.Debounce == "" || r.Debounce == "once_per_utc_day" {
		return "once_per_utc_day"
	}
	if r.Debounce == "every_sync" {
		return "every_sync"
	}
	return r.Debounce
}

func (r RuleSpec) identityKey(repoOrScope string) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%g|%s",
		r.Kind, repoOrScope, r.Metric, r.Window, r.Op, r.Value, r.debounceMode())
}

// milestoneIdentityKey is one debounce key per ladder rung.
func (r RuleSpec) milestoneIdentityKey(threshold int) string {
	return fmt.Sprintf("milestone|%s|stars|%d|once", r.Repo, threshold)
}
