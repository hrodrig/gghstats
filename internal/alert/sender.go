package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hrodrig/gghstats/internal/version"
)

// Sender delivers one Payload to a destination (Slack, webhook, Loki).
// Same shape as pgwd notify.Sender / groot notifier.Sender.
type Sender interface {
	Send(ctx context.Context, p Payload) error
	Type() string
}

// Slack sends via Incoming Webhook using plain text (SPEC §8.6).
type Slack struct {
	WebhookURL string
	Client     *http.Client
}

func (s *Slack) Type() string { return TypeSlack }

func (s *Slack) Send(ctx context.Context, p Payload) error {
	raw, err := json.Marshal(map[string]string{"text": p.CanonicalText()})
	if err != nil {
		return fmt.Errorf("slack payload: %w", err)
	}
	if err := postJSONWithRetryClient(ctx, s.Client, s.WebhookURL, raw, nil); err != nil {
		return fmt.Errorf("slack webhook: %w", err)
	}
	return nil
}

// Webhook POSTs JSON with body presets (generic | discord | teams).
type Webhook struct {
	URL     string
	Headers map[string]string
	Body    string // generic | discord | teams
	Client  *http.Client
}

func (w *Webhook) Type() string { return TypeWebhook }

func (w *Webhook) Send(ctx context.Context, p Payload) error {
	raw, err := webhookJSON(w.Body, p)
	if err != nil {
		return err
	}
	if err := postJSONWithRetryClient(ctx, w.Client, w.URL, raw, func(req *http.Request) {
		for k, v := range w.Headers {
			if strings.TrimSpace(k) == "" {
				continue
			}
			req.Header.Set(k, v)
		}
	}); err != nil {
		return fmt.Errorf("webhook: %w", err)
	}
	return nil
}

func webhookJSON(preset string, p Payload) ([]byte, error) {
	text := p.CanonicalText()
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case BodyDiscord:
		return json.Marshal(map[string]string{"content": text})
	case BodyTeams:
		return json.Marshal(map[string]any{
			"@type":    "MessageCard",
			"@context": "https://schema.org/extensions",
			"summary":  p.OneLine(),
			"text":     text,
		})
	default:
		ver := strings.TrimSpace(p.Version)
		if ver == "" {
			ver = version.Version
		}
		kind := p.Kind
		if kind == "" {
			kind = KindTraffic
		}
		return json.Marshal(map[string]any{
			"source":  "gghstats",
			"version": ver,
			"text":    text,
			"alert": map[string]any{
				"kind":   kind,
				"repo":   p.Repo,
				"scope":  p.Scope,
				"metric": p.Metric,
				"window": p.Window,
				"value":  p.Value,
				"rule":   p.Rule,
				"level":  p.Level,
				"event":  p.Event,
			},
		})
	}
}

// BuildSenders turns resolved sink specs into Sender implementations.
func BuildSenders(sinks []ResolvedSink, client *http.Client) []Sender {
	out := make([]Sender, 0, len(sinks))
	for _, s := range sinks {
		switch s.Type {
		case TypeSlack:
			out = append(out, &Slack{WebhookURL: s.URL, Client: client})
		case TypeWebhook:
			out = append(out, &Webhook{URL: s.URL, Headers: s.Headers, Body: s.Body, Client: client})
		case TypeLoki:
			out = append(out, &Loki{URL: s.URL, Labels: s.Labels, Headers: s.Headers, Client: client})
		case TypeSMTP:
			out = append(out, &SMTP{
				Host:       s.SMTPHost,
				Port:       s.SMTPPort,
				Username:   s.SMTPUser,
				Password:   s.SMTPPassword,
				From:       s.SMTPFrom,
				To:         s.SMTPTo,
				UseTLS:     s.SMTPUseTLS,
				SkipVerify: s.SMTPSkipVerify,
			})
		}
	}
	return out
}

// FanOut sends p to every sender. Continues on error (pgwd/kzero style).
// Returns joined errors if any sink failed; nil if all succeeded or senders empty.
func FanOut(ctx context.Context, senders []Sender, p Payload) error {
	if len(senders) == 0 {
		return fmt.Errorf("no alert senders configured")
	}
	var errs []error
	for _, s := range senders {
		if err := s.Send(ctx, p); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", s.Type(), err))
		}
	}
	return joinErrors(errs)
}

func joinErrors(errs []error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return fmt.Errorf("%s", strings.Join(msgs, "; "))
	}
}
