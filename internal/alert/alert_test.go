package alert

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseSinksJSON_SlackEnv(t *testing.T) {
	getenv := func(k string) string {
		if k == "GGHSTATS_SLACK_WEBHOOK_URL" {
			return "https://hooks.example/slack"
		}
		return ""
	}
	sinks, err := ParseSinksJSON(`[{"type":"slack","webhook_url_env":"GGHSTATS_SLACK_WEBHOOK_URL"}]`, getenv)
	if err != nil {
		t.Fatal(err)
	}
	if len(sinks) != 1 || sinks[0].Type != TypeSlack || sinks[0].URL != "https://hooks.example/slack" {
		t.Fatalf("got %+v", sinks)
	}
}

func TestConfigFromEnv_FailClosed(t *testing.T) {
	getenv := func(k string) string {
		if k == "GGHSTATS_ALERTS_ENABLED" {
			return "true"
		}
		return ""
	}
	_, _, err := ConfigFromEnv(getenv)
	if err == nil {
		t.Fatal("expected error when enabled without sinks")
	}
}

func TestSlackSend(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := &Slack{WebhookURL: srv.URL, Client: srv.Client()}
	p := Payload{
		Kind:    KindTraffic,
		Version: "0.10.0",
		Repo:    "hrodrig/pgwd",
		Metric:  "clones",
		Window:  "1d (UTC)",
		Value:   "241",
		Rule:    "gte 225",
		When:    time.Date(2026, 7, 17, 4, 0, 0, 0, time.UTC),
	}
	if err := s.Send(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(gotBody), &m); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m["text"], "gghstats alert") || !strings.Contains(m["text"], "hrodrig/pgwd") {
		t.Fatalf("unexpected text: %q", m["text"])
	}
}

func TestWebhookDiscordAndGeneric(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		if r.Header.Get("Authorization") != "secret" {
			t.Errorf("missing auth header")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	p := Payload{Kind: KindTraffic, Version: "0.10.0", Repo: "a/b", Metric: "views", Value: "1", When: time.Now().UTC()}
	w := &Webhook{
		URL:     srv.URL,
		Headers: map[string]string{"Authorization": "secret"},
		Body:    BodyDiscord,
		Client:  srv.Client(),
	}
	if err := w.Send(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	w.Body = BodyGeneric
	if err := w.Send(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 2 {
		t.Fatalf("bodies=%d", len(bodies))
	}
	if !strings.Contains(bodies[0], `"content"`) {
		t.Fatalf("discord body: %s", bodies[0])
	}
	if !strings.Contains(bodies[1], `"source":"gghstats"`) {
		t.Fatalf("generic body: %s", bodies[1])
	}
}

func TestLokiSend(t *testing.T) {
	var gotBody string
	var gotOrg string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/loki/api/v1/push") && r.URL.Path != "/" {
			// URL may already include full path when passed to Loki.URL
		}
		gotOrg = r.Header.Get("X-Scope-OrgID")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	l := &Loki{
		URL:     srv.URL,
		Labels:  map[string]string{"job": "gghstats", "source": "alert"},
		Headers: map[string]string{"X-Scope-OrgID": "tenant1"},
		Client:  srv.Client(),
	}
	p := Payload{
		Kind: KindOps, Version: "0.10.0", Level: "warn", Event: "repo_fetch_failed",
		Count: "4", Threshold: "3", Window: "this_sync", When: time.Now().UTC(),
	}
	if err := l.Send(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if gotOrg != "tenant1" {
		t.Fatalf("org=%q", gotOrg)
	}
	var body lokiPushBody
	if err := json.Unmarshal([]byte(gotBody), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Streams) != 1 || body.Streams[0].Stream["job"] != "gghstats" {
		t.Fatalf("streams=%+v", body.Streams)
	}
	if !strings.Contains(body.Streams[0].Values[0][1], "repo_fetch_failed") {
		t.Fatalf("line=%q", body.Streams[0].Values[0][1])
	}
}

func TestFanOut_PartialFailure(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer bad.Close()

	ApplyRetryConfig(RetryConfig{MaxAttempts: 1, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond})
	t.Cleanup(func() { ApplyRetryConfig(DefaultRetryConfig) })

	senders := BuildSenders([]ResolvedSink{
		{Type: TypeSlack, URL: ok.URL},
		{Type: TypeSlack, URL: bad.URL},
	}, ok.Client())
	err := FanOut(context.Background(), senders, Payload{Version: "dev", Repo: "x/y", Metric: "clones", Value: "1"})
	if err == nil {
		t.Fatal("expected partial failure error")
	}
}

func TestNormalizeLokiPushURL(t *testing.T) {
	if got := normalizeLokiPushURL("https://loki.example"); got != "https://loki.example/loki/api/v1/push" {
		t.Fatalf("got %s", got)
	}
	full := "https://loki.example/loki/api/v1/push"
	if got := normalizeLokiPushURL(full); got != full {
		t.Fatalf("got %s", got)
	}
}

func TestHTTPRetryOn5xx(t *testing.T) {
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ApplyRetryConfig(RetryConfig{MaxAttempts: 3, InitialBackoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond})
	t.Cleanup(func() { ApplyRetryConfig(DefaultRetryConfig) })

	s := &Slack{WebhookURL: srv.URL, Client: srv.Client()}
	if err := s.Send(context.Background(), Payload{Version: "dev", Repo: "a/b", Metric: "clones", Value: "1"}); err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("attempts=%d", n)
	}
}
