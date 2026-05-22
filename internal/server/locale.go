package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/hrodrig/gghstats/internal/i18n"
)

const localeCookieMaxAge = 365 * 24 * 3600

type localeBinder struct {
	Locale string
	Lang   string
	T      func(string) string
	Tfmt   func(string, map[string]string) string
}

func newLocaleBinder(locale string, bundle *i18n.Bundle) localeBinder {
	locale = i18n.NormalizeLocale(locale)
	return localeBinder{
		Locale: locale,
		Lang:   i18n.LangAttr(locale),
		T:      func(key string) string { return bundle.T(locale, key) },
		Tfmt:   func(key string, vars map[string]string) string { return bundle.Tfmt(locale, key, vars) },
	}
}

type localeLink struct {
	Code   string
	Label  string
	URL    string
	Active bool
}

func localeFromRequest(r *http.Request, cfg Config) string {
	return i18n.ResolveLocale(r, cfg.DefaultLocale, cfg.EnabledLocales)
}

func maybeSetLocaleCookie(w http.ResponseWriter, r *http.Request, cfg Config) {
	if r.URL.Query().Get("lang") == "" {
		return
	}
	loc := i18n.ResolveLocale(r, cfg.DefaultLocale, cfg.EnabledLocales)
	http.SetCookie(w, &http.Cookie{
		Name:     i18n.CookieName,
		Value:    loc,
		Path:     "/",
		MaxAge:   localeCookieMaxAge,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	})
}

func buildLocaleLinks(r *http.Request, cfg Config, current string) []localeLink {
	labels := map[string]string{
		"en":    "EN",
		"es":    "ES",
		"de":    "DE",
		"fr":    "FR",
		"pt-br": "PT",
	}
	var links []localeLink
	for _, code := range cfg.EnabledLocales {
		code = i18n.NormalizeLocale(code)
		label := labels[code]
		if label == "" {
			label = strings.ToUpper(code)
		}
		links = append(links, localeLink{
			Code:   code,
			Label:  label,
			URL:    localeSwitchURL(r, code),
			Active: code == current,
		})
	}
	return links
}

func localeSwitchURL(r *http.Request, lang string) string {
	q := r.URL.Query()
	q.Set("lang", lang)
	u := url.URL{Path: r.URL.Path, RawQuery: q.Encode()}
	return u.String()
}

func marshalJSI18n(bundle *i18n.Bundle, locale string) template.JS {
	raw, err := json.Marshal(jsI18nPayload(bundle, locale))
	if err != nil {
		return template.JS("{}")
	}
	return template.JS(raw)
}

func mergeLayoutLocale(r *http.Request, cfg Config, data layoutData) layoutData {
	loc := localeFromRequest(r, cfg)
	bundle := i18n.MustLoad()
	lb := newLocaleBinder(loc, bundle)
	data.localeBinder = lb
	data.LocaleLinks = buildLocaleLinks(r, cfg, loc)
	data.JSI18n = marshalJSI18n(bundle, loc)
	if data.PageID == "" && len(data.Breadcrumbs) == 0 {
		data.PageID = "index"
	}
	return data
}

func bindPageLocale(r *http.Request, cfg Config) localeBinder {
	loc := localeFromRequest(r, cfg)
	return newLocaleBinder(loc, i18n.MustLoad())
}

func jsI18nPayload(bundle *i18n.Bundle, locale string) map[string]string {
	keys := []string{
		"js.syncing_all",
		"js.syncing_repo",
		"js.sync_none_yet",
		"js.sync_last_failed",
		"js.sync_last_repo",
		"js.sync_last",
		"js.sync_failed",
		"js.sync_done",
		"js.token_required",
		"js.token_save_sync",
		"common.cancel",
		"common.close_modal",
		"common.theme_light",
		"common.theme_dark",
		"chart.legend_unique",
		"chart.legend_count",
		"chart.legend_clones_count",
		"repo.embed_copied",
	}
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		out[k] = bundle.T(locale, k)
	}
	return out
}
