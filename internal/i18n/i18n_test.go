package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoadAndKeyParity(t *testing.T) {
	b, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	en := b.Keys("en")
	es := b.Keys("es")
	de := b.Keys("de")
	fr := b.Keys("fr")
	pt := b.Keys("pt-br")
	if len(en) == 0 {
		t.Fatal("en has no keys")
	}
	missingES := diffKeys(en, es)
	missingDE := diffKeys(en, de)
	missingFR := diffKeys(en, fr)
	missingPT := diffKeys(en, pt)
	if len(missingES) > 0 {
		t.Errorf("es missing keys: %v", missingES)
	}
	if len(missingDE) > 0 {
		t.Errorf("de missing keys: %v", missingDE)
	}
	if len(missingFR) > 0 {
		t.Errorf("fr missing keys: %v", missingFR)
	}
	if len(missingPT) > 0 {
		t.Errorf("pt-br missing keys: %v", missingPT)
	}
}

func diffKeys(want, have []string) []string {
	set := make(map[string]struct{}, len(have))
	for _, k := range have {
		set[k] = struct{}{}
	}
	var missing []string
	for _, k := range want {
		if _, ok := set[k]; !ok {
			missing = append(missing, k)
		}
	}
	return missing
}

func TestResolveLocale(t *testing.T) {
	enabled := []string{"en", "es", "de"}
	defaultLoc := "en"

	r := httptest.NewRequest("GET", "/?lang=de", nil)
	if got := ResolveLocale(r, defaultLoc, enabled); got != "de" {
		t.Fatalf("query: got %q", got)
	}

	r = httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: "es"})
	if got := ResolveLocale(r, defaultLoc, enabled); got != "es" {
		t.Fatalf("cookie: got %q", got)
	}

	r = httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Language", "de-DE,en;q=0.9")
	if got := ResolveLocale(r, defaultLoc, enabled); got != "de" {
		t.Fatalf("accept: got %q", got)
	}

	r = httptest.NewRequest("GET", "/?lang=fr", nil)
	if got := ResolveLocale(r, defaultLoc, enabled); got != "en" {
		t.Fatalf("unsupported query falls back to en: got %q", got)
	}
}

func TestTfmt(t *testing.T) {
	b, _ := Load()
	got := b.Tfmt("en", "index.showing", map[string]string{"from": "1", "to": "10", "total": "50"})
	want := "Showing 1–10 of 50 repositories"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
