package alert

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// SinkType values supported in 0.10.0 MVP.
const (
	TypeSlack   = "slack"
	TypeWebhook = "webhook"
	TypeLoki    = "loki"
)

// Body presets for type=webhook.
const (
	BodyGeneric = "generic"
	BodyDiscord = "discord"
	BodyTeams   = "teams"
)

// SinkSpec is the JSON shape of one entry in GGHSTATS_ALERT_SINKS.
type SinkSpec struct {
	Type          string            `json:"type"`
	WebhookURLEnv string            `json:"webhook_url_env,omitempty"`
	WebhookURL    string            `json:"webhook_url,omitempty"` // dev-only / tests
	URLEnv        string            `json:"url_env,omitempty"`
	URL           string            `json:"url,omitempty"` // dev-only / tests
	HeadersEnv    map[string]string `json:"headers_env,omitempty"`
	Body          string            `json:"body,omitempty"`   // discord | teams | generic
	Labels        map[string]string `json:"labels,omitempty"` // loki
}

// ResolvedSink is a sink ready to deliver (secrets resolved from the environment).
type ResolvedSink struct {
	Type    string
	URL     string
	Headers map[string]string
	Body    string
	Labels  map[string]string
}

// ParseSinksJSON parses the GGHSTATS_ALERT_SINKS JSON array and resolves *_env fields.
func ParseSinksJSON(raw string, getenv func(string) string) ([]ResolvedSink, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if getenv == nil {
		getenv = os.Getenv
	}
	var specs []SinkSpec
	if err := json.Unmarshal([]byte(raw), &specs); err != nil {
		return nil, fmt.Errorf("GGHSTATS_ALERT_SINKS: %w", err)
	}
	out := make([]ResolvedSink, 0, len(specs))
	for i, s := range specs {
		r, err := resolveSink(s, getenv)
		if err != nil {
			return nil, fmt.Errorf("GGHSTATS_ALERT_SINKS[%d]: %w", i, err)
		}
		out = append(out, r)
	}
	return out, nil
}

func resolveSink(s SinkSpec, getenv func(string) string) (ResolvedSink, error) {
	t := strings.ToLower(strings.TrimSpace(s.Type))
	switch t {
	case TypeSlack:
		url, err := resolveURL(s.WebhookURLEnv, s.WebhookURL, getenv, "webhook_url_env")
		if err != nil {
			return ResolvedSink{}, err
		}
		return ResolvedSink{Type: TypeSlack, URL: url, Headers: resolveHeaders(s.HeadersEnv, getenv)}, nil
	case TypeWebhook:
		url, err := resolveURL(s.URLEnv, s.URL, getenv, "url_env")
		if err != nil {
			return ResolvedSink{}, err
		}
		body := strings.ToLower(strings.TrimSpace(s.Body))
		if body == "" {
			body = BodyGeneric
		}
		switch body {
		case BodyGeneric, BodyDiscord, BodyTeams:
		default:
			return ResolvedSink{}, fmt.Errorf("webhook body %q not supported (use generic, discord, or teams)", s.Body)
		}
		return ResolvedSink{
			Type:    TypeWebhook,
			URL:     url,
			Headers: resolveHeaders(s.HeadersEnv, getenv),
			Body:    body,
		}, nil
	case TypeLoki:
		url, err := resolveURL(s.URLEnv, s.URL, getenv, "url_env")
		if err != nil {
			return ResolvedSink{}, err
		}
		url = normalizeLokiPushURL(url)
		labels := map[string]string{"job": "gghstats"}
		for k, v := range s.Labels {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			labels[k] = v
		}
		if _, ok := labels["job"]; !ok || labels["job"] == "" {
			labels["job"] = "gghstats"
		}
		return ResolvedSink{
			Type:    TypeLoki,
			URL:     url,
			Headers: resolveHeaders(s.HeadersEnv, getenv),
			Labels:  labels,
		}, nil
	case "":
		return ResolvedSink{}, fmt.Errorf("type is required")
	default:
		return ResolvedSink{}, fmt.Errorf("unsupported type %q (use slack, webhook, or loki)", s.Type)
	}
}

func resolveURL(envName, inline string, getenv func(string) string, envField string) (string, error) {
	if strings.TrimSpace(envName) != "" {
		v := strings.TrimSpace(getenv(envName))
		if v == "" {
			return "", fmt.Errorf("%s %q is empty", envField, envName)
		}
		return v, nil
	}
	inline = strings.TrimSpace(inline)
	if inline == "" {
		return "", fmt.Errorf("missing %s or inline url", envField)
	}
	return inline, nil
}

func resolveHeaders(headersEnv map[string]string, getenv func(string) string) map[string]string {
	if len(headersEnv) == 0 {
		return nil
	}
	out := make(map[string]string, len(headersEnv))
	for hdr, envName := range headersEnv {
		hdr = strings.TrimSpace(hdr)
		envName = strings.TrimSpace(envName)
		if hdr == "" || envName == "" {
			continue
		}
		out[hdr] = getenv(envName)
	}
	return out
}

func normalizeLokiPushURL(u string) string {
	u = strings.TrimRight(strings.TrimSpace(u), "/")
	if strings.HasSuffix(u, "/loki/api/v1/push") {
		return u
	}
	return u + "/loki/api/v1/push"
}

// ConfigFromEnv loads alerts enable flag and resolved sinks (SPEC §8.5 fail-closed).
func ConfigFromEnv(getenv func(string) string) (enabled bool, sinks []ResolvedSink, err error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	enabled = envBool(getenv("GGHSTATS_ALERTS_ENABLED"), false)
	sinks, err = ParseSinksJSON(getenv("GGHSTATS_ALERT_SINKS"), getenv)
	if err != nil {
		return enabled, nil, err
	}
	if enabled && len(sinks) == 0 {
		return true, nil, fmt.Errorf("GGHSTATS_ALERTS_ENABLED=true but GGHSTATS_ALERT_SINKS is empty or invalid")
	}
	return enabled, sinks, nil
}

func envBool(v string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	case "":
		return def
	default:
		return def
	}
}
