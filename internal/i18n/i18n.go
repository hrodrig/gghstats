// Package i18n loads UI locale JSON and resolves locale from request.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
)

//go:embed locales/*.json
var localeFS embed.FS

const (
	DefaultLocale = "en"
	CookieName    = "gghstats_locale"
)

// Bundle holds flattened translations per locale.
type Bundle struct {
	byLocale map[string]map[string]string
	fallback map[string]string
}

var (
	global     *Bundle
	globalOnce sync.Once
	globalErr  error
)

// MustLoad returns the global bundle, loading on first call.
func MustLoad() *Bundle {
	globalOnce.Do(func() {
		global, globalErr = Load()
	})
	if globalErr != nil {
		panic(globalErr)
	}
	return global
}

// Load reads all embedded locale files.
func Load() (*Bundle, error) {
	entries, err := localeFS.ReadDir("locales")
	if err != nil {
		return nil, err
	}
	b := &Bundle{byLocale: make(map[string]map[string]string)}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		loc := strings.TrimSuffix(e.Name(), ".json")
		raw, err := localeFS.ReadFile("locales/" + e.Name())
		if err != nil {
			return nil, err
		}
		flat, err := flattenJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("locale %s: %w", loc, err)
		}
		b.byLocale[loc] = flat
	}
	if fb, ok := b.byLocale[DefaultLocale]; ok {
		b.fallback = fb
	}
	return b, nil
}

func flattenJSON(raw []byte) (map[string]string, error) {
	var nested map[string]json.RawMessage
	if err := json.Unmarshal(raw, &nested); err != nil {
		return nil, err
	}
	out := make(map[string]string)
	var walk func(prefix string, m map[string]json.RawMessage)
	walk = func(prefix string, m map[string]json.RawMessage) {
		for k, v := range m {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			if len(v) > 0 && v[0] == '{' {
				var inner map[string]json.RawMessage
				if json.Unmarshal(v, &inner) == nil {
					walk(key, inner)
					continue
				}
			}
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				continue
			}
			out[key] = s
		}
	}
	walk("", nested)
	return out, nil
}

// T returns the translation for key in locale, falling back to English then the key.
func (b *Bundle) T(locale, key string) string {
	locale = NormalizeLocale(locale)
	if m, ok := b.byLocale[locale]; ok {
		if s, ok := m[key]; ok && s != "" {
			return s
		}
	}
	if s, ok := b.fallback[key]; ok {
		return s
	}
	return key
}

// Tfmt replaces {{name}} placeholders in the translated string.
func (b *Bundle) Tfmt(locale, key string, vars map[string]string) string {
	s := b.T(locale, key)
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	return s
}

// Keys returns sorted keys for a locale (for tests).
func (b *Bundle) Keys(locale string) []string {
	m := b.byLocale[NormalizeLocale(locale)]
	if m == nil {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// NormalizeLocale maps tags like es-ES to es, pt-BR to pt-br.
func NormalizeLocale(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return DefaultLocale
	}
	if i := strings.IndexByte(s, ','); i >= 0 {
		s = s[:i]
	}
	if i := strings.IndexByte(s, ';'); i >= 0 {
		s = s[:i]
	}
	if tag, region, ok := strings.Cut(s, "-"); ok {
		if strings.EqualFold(tag, "pt") && strings.EqualFold(region, "br") {
			return "pt-br"
		}
		return tag
	}
	return s
}

// ParseEnabledLocales parses GGHSTATS_ENABLED_LOCALES (comma-separated).
func ParseEnabledLocales(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{"en", "es", "de"}
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = NormalizeLocale(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	if len(out) == 0 {
		return []string{DefaultLocale}
	}
	return out
}

// ParseDefaultLocale reads GGHSTATS_DEFAULT_LOCALE.
func ParseDefaultLocale(s string) string {
	s = NormalizeLocale(s)
	if s == "" {
		return DefaultLocale
	}
	return s
}

// ResolveLocale picks locale: ?lang= → cookie → Accept-Language → default.
func ResolveLocale(r *http.Request, defaultLocale string, enabled []string) string {
	defaultLocale = NormalizeLocale(defaultLocale)
	enabledSet := make(map[string]struct{}, len(enabled))
	for _, e := range enabled {
		enabledSet[NormalizeLocale(e)] = struct{}{}
	}
	pick := func(loc string) string {
		loc = NormalizeLocale(loc)
		if _, ok := enabledSet[loc]; ok {
			return loc
		}
		if _, ok := enabledSet[DefaultLocale]; ok {
			return DefaultLocale
		}
		for e := range enabledSet {
			return e
		}
		return defaultLocale
	}

	if q := r.URL.Query().Get("lang"); q != "" {
		return pick(q)
	}
	if c, err := r.Cookie(CookieName); err == nil && c.Value != "" {
		return pick(c.Value)
	}
	if al := r.Header.Get("Accept-Language"); al != "" {
		for _, part := range strings.Split(al, ",") {
			tag, _, _ := strings.Cut(strings.TrimSpace(part), ";")
			if tag != "" {
				return pick(tag)
			}
		}
	}
	return pick(defaultLocale)
}

// LangAttr returns BCP 47 lang for <html lang="…">.
func LangAttr(locale string) string {
	locale = NormalizeLocale(locale)
	switch locale {
	case "en":
		return "en"
	case "es":
		return "es"
	case "de":
		return "de"
	case "fr":
		return "fr"
	case "pt-br":
		return "pt-BR"
	default:
		return locale
	}
}

// EnvDefaultLocale reads GGHSTATS_DEFAULT_LOCALE from the environment.
func EnvDefaultLocale() string {
	return ParseDefaultLocale(os.Getenv("GGHSTATS_DEFAULT_LOCALE"))
}

// EnvEnabledLocales reads GGHSTATS_ENABLED_LOCALES.
func EnvEnabledLocales() []string {
	return ParseEnabledLocales(os.Getenv("GGHSTATS_ENABLED_LOCALES"))
}
