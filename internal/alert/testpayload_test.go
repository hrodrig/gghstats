package alert

import (
	"strings"
	"testing"
)

func TestSyntheticPayload(t *testing.T) {
	p, err := SyntheticPayload(KindTraffic)
	if err != nil {
		t.Fatal(err)
	}
	text := p.CanonicalText()
	if !strings.Contains(text, "delivery check") || !strings.Contains(text, "example/repo") {
		t.Fatalf("traffic payload: %s", text)
	}
	p, err = SyntheticPayload(KindOps)
	if err != nil {
		t.Fatal(err)
	}
	text = p.CanonicalText()
	if !strings.Contains(text, "alert_test") || !strings.Contains(text, "level:") {
		t.Fatalf("ops payload: %s", text)
	}
	if _, err := SyntheticPayload("nope"); err == nil {
		t.Fatal("expected unknown kind error")
	}
}

func TestFilterSinks(t *testing.T) {
	sinks := []ResolvedSink{
		{Type: TypeSlack, URL: "http://s"},
		{Type: TypeLoki, URL: "http://l"},
	}
	got, err := FilterSinks(sinks, "loki")
	if err != nil || len(got) != 1 || got[0].Type != TypeLoki {
		t.Fatalf("got %+v err=%v", got, err)
	}
	if _, err := FilterSinks(sinks, "webhook"); err == nil {
		t.Fatal("expected no webhook sink error")
	}
}

func TestSinksForTest(t *testing.T) {
	getenv := func(k string) string {
		if k == "GGHSTATS_ALERT_SINKS" {
			return `[{"type":"slack","webhook_url":"https://hooks.example/x"}]`
		}
		return ""
	}
	sinks, err := SinksForTest(getenv)
	if err != nil || len(sinks) != 1 {
		t.Fatalf("got %+v err=%v", sinks, err)
	}
	// ENABLED not required
	if _, err := SinksForTest(func(string) string { return "" }); err == nil {
		t.Fatal("expected empty sinks error")
	}
}
