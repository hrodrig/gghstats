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
		writeOpsBody(&b, p, whenStr)
	} else {
		writeTrafficBody(&b, p, whenStr)
	}
	return strings.TrimRight(b.String(), "\n")
}

func writeField(b *strings.Builder, label, value string) {
	if value == "" {
		return
	}
	b.WriteString(label)
	b.WriteString(value)
	b.WriteByte('\n')
}

func writeOpsBody(b *strings.Builder, p Payload, whenStr string) {
	writeField(b, "level:    ", p.Level)
	writeField(b, "event:    ", p.Event)
	writeField(b, "count:    ", p.Count)
	writeField(b, "threshold: ", p.Threshold)
	writeField(b, "window:   ", p.Window)
	writeField(b, "detail:   ", p.Detail)
	b.WriteString("when:     " + whenStr + "\n")
}

func writeTrafficBody(b *strings.Builder, p Payload, whenStr string) {
	writeField(b, "scope:    ", p.Scope)
	writeField(b, "repo:     ", p.Repo)
	writeField(b, "metric:   ", p.Metric)
	writeField(b, "window:   ", p.Window)
	writeField(b, "value:    ", p.Value)
	writeField(b, "rule:     ", p.Rule)
	b.WriteString("when:     " + whenStr + "\n")
	writeField(b, "dash:     ", p.Dash)
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
