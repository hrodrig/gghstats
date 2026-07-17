package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Loki pushes alert lines to Grafana Loki's push API (pgwd pattern).
type Loki struct {
	URL     string
	Labels  map[string]string
	Headers map[string]string // e.g. X-Scope-OrgID, Authorization
	Client  *http.Client
}

func (l *Loki) Type() string { return TypeLoki }

type lokiPushBody struct {
	Streams []lokiStream `json:"streams"`
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

// PushPayload returns the JSON body Send posts. Useful for tests.
func (l *Loki) PushPayload(p Payload) ([]byte, error) {
	labels := l.buildLabels(p)
	line := p.CanonicalText()
	when := p.When
	if when.IsZero() {
		when = time.Now().UTC()
	}
	ts := strconv.FormatInt(when.UnixNano(), 10)
	body := lokiPushBody{
		Streams: []lokiStream{{
			Stream: labels,
			Values: [][]string{{ts, line}},
		}},
	}
	return json.Marshal(body)
}

func (l *Loki) buildLabels(p Payload) map[string]string {
	labels := make(map[string]string)
	for k, v := range l.Labels {
		labels[k] = v
	}
	if labels["job"] == "" {
		labels["job"] = "gghstats"
	}
	if p.Kind != "" {
		labels["kind"] = p.Kind
	}
	if p.Level != "" {
		labels["level"] = p.Level
	}
	if p.Event != "" {
		labels["event"] = p.Event
	}
	if p.Metric != "" {
		labels["metric"] = p.Metric
	}
	return labels
}

// Send pushes one log line to Loki.
func (l *Loki) Send(ctx context.Context, p Payload) error {
	raw, err := l.PushPayload(p)
	if err != nil {
		return err
	}
	if err := postJSONWithRetryClient(ctx, l.Client, l.URL, raw, func(req *http.Request) {
		for k, v := range l.Headers {
			if strings.TrimSpace(k) == "" || v == "" {
				continue
			}
			req.Header.Set(k, v)
		}
	}); err != nil {
		return fmt.Errorf("loki push: %w", err)
	}
	return nil
}
