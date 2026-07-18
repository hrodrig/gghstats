package alert

import (
	"fmt"
	"strings"
	"time"

	"github.com/hrodrig/gghstats/internal/version"
)

// SyntheticPayload builds a delivery-check Payload for `gghstats alert test` (SPEC §8.8).
func SyntheticPayload(kind string) (Payload, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		kind = KindTraffic
	}
	now := time.Now().UTC()
	switch kind {
	case KindTraffic:
		return Payload{
			Kind:    KindTraffic,
			Version: version.Version,
			When:    now,
			Repo:    "example/repo",
			Metric:  "clones",
			Window:  "1d (UTC)",
			Value:   "0",
			Rule:    "delivery check",
		}, nil
	case KindOps:
		return Payload{
			Kind:    KindOps,
			Version: version.Version,
			When:    now,
			Level:   "info",
			Event:   "alert_test",
			Window:  "n/a",
			Detail:  "delivery check (gghstats alert test)",
		}, nil
	default:
		return Payload{}, fmt.Errorf("unknown kind %q (want: traffic, ops)", kind)
	}
}

// FilterSinks keeps sinks whose Type matches sinkType (empty = all).
func FilterSinks(sinks []ResolvedSink, sinkType string) ([]ResolvedSink, error) {
	sinkType = strings.ToLower(strings.TrimSpace(sinkType))
	if sinkType == "" {
		return sinks, nil
	}
	switch sinkType {
	case TypeSlack, TypeWebhook, TypeLoki, TypeSMTP:
	default:
		return nil, fmt.Errorf("unknown sink type %q (want: slack, webhook, loki, smtp)", sinkType)
	}
	out := make([]ResolvedSink, 0, len(sinks))
	for _, s := range sinks {
		if s.Type == sinkType {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no %q sink in GGHSTATS_ALERT_SINKS", sinkType)
	}
	return out, nil
}

// SinksForTest loads sinks for smoke-test. Does not require GGHSTATS_ALERTS_ENABLED
// (ENABLED gates post-sync evaluation only — SPEC §8.8).
func SinksForTest(getenv func(string) string) ([]ResolvedSink, error) {
	sinks, err := ParseSinksJSON(getenv("GGHSTATS_ALERT_SINKS"), getenv)
	if err != nil {
		return nil, err
	}
	if len(sinks) == 0 {
		return nil, fmt.Errorf("GGHSTATS_ALERT_SINKS is empty; configure at least one sink")
	}
	return sinks, nil
}
