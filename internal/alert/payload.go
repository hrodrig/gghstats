// Package alert delivers opt-in operator notifications (SPEC §8).
// Sinks ship first; rule evaluation builds on Deliverer.
package alert

import (
	"fmt"
	"strings"
	"time"

	"github.com/hrodrig/gghstats/internal/version"
)

// KindTraffic and KindOps distinguish payload shapes in sinks and Loki labels.
const (
	KindTraffic = "traffic"
	KindOps     = "ops"
)

// Payload is one alert notification. Text is built via CanonicalText when empty.
type Payload struct {
	Kind    string // traffic | ops
	Version string // defaults to version.Version
	When    time.Time

	// Traffic fields
	Repo   string
	Scope  string
	Metric string
	Window string
	Value  string
	Rule   string
	Dash   string

	// Ops fields
	Level     string
	Event     string
	Count     string
	Threshold string
	Detail    string
}

// CanonicalText returns the multiline plain-text body (SPEC §8.6).
func (p Payload) CanonicalText() string {
	ver := strings.TrimSpace(p.Version)
	if ver == "" {
		ver = version.Version
	}
	when := p.When
	if when.IsZero() {
		when = time.Now().UTC()
	}
	whenStr := when.UTC().Format(time.RFC3339)

	var b strings.Builder
	b.WriteString("gghstats alert\n")
	b.WriteString("version:  " + ver + "\n")
	if p.Kind == KindOps || p.Kind == "ops" {
		if p.Level != "" {
			b.WriteString("level:    " + p.Level + "\n")
		}
		if p.Event != "" {
			b.WriteString("event:    " + p.Event + "\n")
		}
		if p.Count != "" {
			b.WriteString("count:    " + p.Count + "\n")
		}
		if p.Threshold != "" {
			b.WriteString("threshold: " + p.Threshold + "\n")
		}
		if p.Window != "" {
			b.WriteString("window:   " + p.Window + "\n")
		}
		if p.Detail != "" {
			b.WriteString("detail:   " + p.Detail + "\n")
		}
		b.WriteString("when:     " + whenStr + "\n")
		return strings.TrimRight(b.String(), "\n")
	}

	if p.Scope != "" {
		b.WriteString("scope:    " + p.Scope + "\n")
	}
	if p.Repo != "" {
		b.WriteString("repo:     " + p.Repo + "\n")
	}
	if p.Metric != "" {
		b.WriteString("metric:   " + p.Metric + "\n")
	}
	if p.Window != "" {
		b.WriteString("window:   " + p.Window + "\n")
	}
	if p.Value != "" {
		b.WriteString("value:    " + p.Value + "\n")
	}
	if p.Rule != "" {
		b.WriteString("rule:     " + p.Rule + "\n")
	}
	b.WriteString("when:     " + whenStr + "\n")
	if p.Dash != "" {
		b.WriteString("dash:     " + p.Dash + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// OneLine is a short fallback title for Slack/logs.
func (p Payload) OneLine() string {
	ver := strings.TrimSpace(p.Version)
	if ver == "" {
		ver = version.Version
	}
	if p.Kind == KindOps || p.Level != "" {
		return fmt.Sprintf("gghstats/%s ops %s: %s", ver, p.Level, p.Event)
	}
	target := p.Repo
	if target == "" {
		target = p.Scope
	}
	return fmt.Sprintf("gghstats/%s alert: %s %s = %s", ver, target, p.Metric, p.Value)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
