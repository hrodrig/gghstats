package alert

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// SinkType values supported for GGHSTATS_ALERT_SINKS.
const (
	TypeSlack   = "slack"
	TypeWebhook = "webhook"
	TypeLoki    = "loki"
	TypeSMTP    = "smtp"
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

	// SMTP (SPEC §8.5) — secrets via *_env; inline fields for tests only.
	HostEnv     string `json:"host_env,omitempty"`
	PortEnv     string `json:"port_env,omitempty"`
	UserEnv     string `json:"user_env,omitempty"`
	PasswordEnv string `json:"password_env,omitempty"`
	FromEnv     string `json:"from_env,omitempty"`
	ToEnv       string `json:"to_env,omitempty"`
	Host        string `json:"host,omitempty"`
	Port        string `json:"port,omitempty"` // string so JSON can be "587"
	User        string `json:"user,omitempty"`
	Password    string `json:"password,omitempty"`
	From        string `json:"from,omitempty"`
	To          string `json:"to,omitempty"`
	UseTLS      bool   `json:"use_tls,omitempty"`     // implicit TLS (e.g. 465)
	SkipVerify  bool   `json:"skip_verify,omitempty"` // insecure; tests / lab only
}

// ResolvedSink is a sink ready to deliver (secrets resolved from the environment).
type ResolvedSink struct {
	Type    string
	URL     string
	Headers map[string]string
	Body    string
	Labels  map[string]string

	// SMTP
	SMTPHost       string
	SMTPPort       int
	SMTPUser       string
	SMTPPassword   string
	SMTPFrom       string
	SMTPTo         []string
	SMTPUseTLS     bool
	SMTPSkipVerify bool
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
	case TypeSMTP:
		return resolveSMTPSink(s, getenv)
	case "":
		return ResolvedSink{}, fmt.Errorf("type is required")
	default:
		return ResolvedSink{}, fmt.Errorf("unsupported type %q (use slack, webhook, loki, or smtp)", s.Type)
	}
}

func resolveSMTPSink(s SinkSpec, getenv func(string) string) (ResolvedSink, error) {
	host, err := resolveOptional(s.HostEnv, s.Host, getenv, "host_env")
	if err != nil {
		return ResolvedSink{}, err
	}
	if host == "" {
		return ResolvedSink{}, fmt.Errorf("smtp host_env or host is required")
	}
	portStr, err := resolveOptional(s.PortEnv, s.Port, getenv, "port_env")
	if err != nil {
		return ResolvedSink{}, err
	}
	port := 587
	if strings.TrimSpace(portStr) != "" {
		p, err := strconv.Atoi(strings.TrimSpace(portStr))
		if err != nil || p <= 0 || p > 65535 {
			return ResolvedSink{}, fmt.Errorf("smtp port %q invalid", portStr)
		}
		port = p
	}
	user, err := resolveOptional(s.UserEnv, s.User, getenv, "user_env")
	if err != nil {
		return ResolvedSink{}, err
	}
	password, err := resolveOptional(s.PasswordEnv, s.Password, getenv, "password_env")
	if err != nil {
		return ResolvedSink{}, err
	}
	from, err := resolveOptional(s.FromEnv, s.From, getenv, "from_env")
	if err != nil {
		return ResolvedSink{}, err
	}
	if from == "" {
		from = user
	}
	if from == "" {
		return ResolvedSink{}, fmt.Errorf("smtp from_env/from or user_env/user is required")
	}
	toRaw, err := resolveOptional(s.ToEnv, s.To, getenv, "to_env")
	if err != nil {
		return ResolvedSink{}, err
	}
	to := splitRecipients(toRaw)
	if len(to) == 0 {
		return ResolvedSink{}, fmt.Errorf("smtp to_env or to is required")
	}
	return ResolvedSink{
		Type:           TypeSMTP,
		SMTPHost:       host,
		SMTPPort:       port,
		SMTPUser:       user,
		SMTPPassword:   password,
		SMTPFrom:       from,
		SMTPTo:         to,
		SMTPUseTLS:     s.UseTLS,
		SMTPSkipVerify: s.SkipVerify,
	}, nil
}

// resolveOptional returns env value when envName set; empty env name uses inline.
// Empty result is OK (caller decides required fields).
func resolveOptional(envName, inline string, getenv func(string) string, envField string) (string, error) {
	if strings.TrimSpace(envName) != "" {
		v := strings.TrimSpace(getenv(envName))
		if v == "" {
			return "", fmt.Errorf("%s %q is empty", envField, envName)
		}
		return v, nil
	}
	return strings.TrimSpace(inline), nil
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
