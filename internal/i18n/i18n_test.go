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

func TestGermanH2HTitleLocalized(t *testing.T) {
	b, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	got := b.T("de", "h2h.title")
	want := "Direktvergleich (H2H)"
	if got != want {
		t.Fatalf("de h2h.title: got %q want %q", got, want)
	}
	if got == b.T("en", "h2h.title") {
		t.Fatal("de h2h.title must not match English")
	}
}

func TestMustLoad(t *testing.T) {
	b := MustLoad()
	if b.T("en", "nav.home") == "nav.home" {
		t.Fatal("MustLoad bundle should resolve keys")
	}
}

func TestNormalizeLocale(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", DefaultLocale},
		{"  de-DE  ", "de"},
		{"pt-BR", "pt-br"},
		{"PT-br", "pt-br"},
		{"es,en;q=0.8", "es"},
		{"fr;foo", "fr"},
	}
	for _, tc := range tests {
		if got := NormalizeLocale(tc.in); got != tc.want {
			t.Fatalf("NormalizeLocale(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseEnabledAndDefaultLocales(t *testing.T) {
	if got := ParseEnabledLocales(""); len(got) != 3 {
		t.Fatalf("empty enabled: %v", got)
	}
	got := ParseEnabledLocales(" fr , pt-br ")
	if len(got) != 2 || got[0] != "fr" || got[1] != "pt-br" {
		t.Fatalf("got %v", got)
	}
	if ParseDefaultLocale("") != DefaultLocale {
		t.Fatal("default empty")
	}
	if ParseDefaultLocale("DE") != "de" {
		t.Fatal("default de")
	}
}

func TestTFallback(t *testing.T) {
	b, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := b.T("en", "no.such.key"); got != "no.such.key" {
		t.Fatalf("missing key: %q", got)
	}
	if got := b.T("xx", "nav.home"); got == "xx" {
		t.Fatalf("unknown locale should fall back to en: %q", got)
	}
}

func TestLangAttr(t *testing.T) {
	if LangAttr("pt-br") != "pt-BR" {
		t.Fatal("pt-br")
	}
	if LangAttr("de") != "de" {
		t.Fatal("de")
	}
	if LangAttr("zz") != "zz" {
		t.Fatal("default passthrough")
	}
}

func TestEnvLocaleHelpers(t *testing.T) {
	t.Setenv("GGHSTATS_DEFAULT_LOCALE", "de")
	if got := EnvDefaultLocale(); got != "de" {
		t.Fatalf("default: %q", got)
	}
	t.Setenv("GGHSTATS_ENABLED_LOCALES", "en,fr")
	if got := EnvEnabledLocales(); len(got) != 2 || got[1] != "fr" {
		t.Fatalf("enabled: %v", got)
	}
}

func TestResolveLocaleFallbackEnabled(t *testing.T) {
	enabled := []string{"de"}
	r := httptest.NewRequest("GET", "/?lang=fr", nil)
	got := ResolveLocale(r, "de", enabled)
	if got != "de" {
		t.Fatalf("unsupported lang picks enabled locale: got %q", got)
	}

	r = httptest.NewRequest("GET", "/", nil)
	got = ResolveLocale(r, "de", enabled)
	if got != "de" {
		t.Fatalf("default: got %q", got)
	}
}
