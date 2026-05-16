package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hrodrig/gghstats/internal/store"
)

const defaultBadgeCacheMaxAge = 300

type badgeMetric int

const (
	badgeMetricClones badgeMetric = iota
	badgeMetricClones30d
	badgeMetricViews
	badgeMetricStars
)

func parseBadgeMetric(raw string) (badgeMetric, bool) {
	switch strings.TrimSpace(raw) {
	case "", "clones":
		return badgeMetricClones, true
	case "clones_30d":
		return badgeMetricClones30d, true
	case "views":
		return badgeMetricViews, true
	case "stars":
		return badgeMetricStars, true
	default:
		return 0, false
	}
}

func defaultBadgeLabel(m badgeMetric) string {
	switch m {
	case badgeMetricClones30d:
		return "clones 30d"
	case badgeMetricViews:
		return "views"
	case badgeMetricStars:
		return "stars"
	default:
		return "clones"
	}
}

func badgeValue(summary *store.RepoSummary, m badgeMetric) int {
	switch m {
	case badgeMetricClones30d:
		return summary.Clones30d
	case badgeMetricViews:
		return summary.TotalViews
	case badgeMetricStars:
		return summary.Stars
	default:
		return summary.TotalClones
	}
}

func badgeOwnerRepo(r *http.Request) string {
	owner := r.PathValue("owner")
	repo := strings.TrimSuffix(r.PathValue("repo"), ".svg")
	if owner == "" || repo == "" {
		return ""
	}
	return owner + "/" + repo
}

func publicBaseURL(r *http.Request, configured string) string {
	if s := strings.TrimRight(strings.TrimSpace(configured), "/"); s != "" {
		return s
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	} else if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = strings.ToLower(strings.TrimSpace(strings.Split(proto, ",")[0]))
	}
	return scheme + "://" + r.Host
}

func badgeURL(base, fullName string, m badgeMetric, customLabel string) string {
	u, err := url.Parse(base + "/api/v1/badge/" + fullName)
	if err != nil {
		return base + "/api/v1/badge/" + fullName
	}
	q := u.Query()
	switch m {
	case badgeMetricClones30d:
		q.Set("metric", "clones_30d")
	case badgeMetricViews:
		q.Set("metric", "views")
	case badgeMetricStars:
		q.Set("metric", "stars")
	default:
		q.Set("metric", "clones")
	}
	if customLabel != "" && customLabel != defaultBadgeLabel(m) {
		q.Set("label", customLabel)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func badgeMiddleware(cfg Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.BadgePublic {
			next(w, r)
			return
		}
		if cfg.APIToken == "" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("x-api-token") != cfg.APIToken {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func handleBadge(cfg Config, db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fullName := badgeOwnerRepo(r)
		if fullName == "" {
			writeBadgeError(w, r, http.StatusBadRequest, "invalid path")
			return
		}

		metric, ok := parseBadgeMetric(r.URL.Query().Get("metric"))
		if !ok {
			writeBadgeError(w, r, http.StatusBadRequest, "invalid metric")
			return
		}

		label := strings.TrimSpace(r.URL.Query().Get("label"))
		if label == "" {
			label = defaultBadgeLabel(metric)
		}

		style := strings.TrimSpace(r.URL.Query().Get("style"))
		if style == "" {
			style = "flat"
		}
		if style != "flat" && style != "flat-square" {
			writeBadgeError(w, r, http.StatusBadRequest, "invalid style")
			return
		}

		summary, err := db.RepoByName(fullName)
		if err != nil {
			writeBadgeError(w, r, http.StatusInternalServerError, "database error")
			return
		}
		if summary == nil {
			writeBadgeNotFound(w, r)
			return
		}

		value := formatBadgeNumber(badgeValue(summary, metric))
		svg := renderBadgeSVG(label, value, style)
		maxAge := cfg.BadgeCacheMaxAge
		if maxAge <= 0 {
			maxAge = defaultBadgeCacheMaxAge
		}
		w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
		if _, err := w.Write(svg); err != nil {
			slog.Error("write badge svg", "error", err)
		}
	}
}

func wantsJSONResponse(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

func writeBadgeError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	if wantsJSONResponse(r) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		http.Error(w, fmt.Sprintf(`{"error":%q}`, msg), code)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.WriteHeader(code)
	_, _ = w.Write(renderBadgeSVG("error", msg, "flat"))
}

func writeBadgeNotFound(w http.ResponseWriter, r *http.Request) {
	if wantsJSONResponse(r) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":"not_found"}`)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write(renderBadgeSVG("unknown", "n/a", "flat"))
}

func formatBadgeNumber(n int) string {
	s := strconv.FormatInt(int64(n), 10)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}

func svgEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

// renderBadgeSVG returns a minimal shields.io-style flat badge (no external deps).
func renderBadgeSVG(label, value, style string) []byte {
	label = svgEscape(label)
	value = svgEscape(value)
	labelW := len(label)*7 + 10
	if labelW < 54 {
		labelW = 54
	}
	valueW := len(value)*7 + 10
	if valueW < 40 {
		valueW = 40
	}
	totalW := labelW + valueW
	rx := 3
	if style == "flat-square" {
		rx = 0
	}
	labelX := labelW / 2
	valueX := labelW + valueW/2
	return []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s: %s"><title>%s: %s</title><linearGradient id="s" x2="0" y2="100%%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="r"><rect width="%d" height="20" rx="%d" fill="#fff"/></clipPath><g clip-path="url(#r)"><rect width="%d" height="20" fill="#555"/><rect x="%d" width="%d" height="20" fill="#4c1"/><rect width="%d" height="20" fill="url(#s)"/><g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="11"><text x="%d" y="15" fill="#fff">%s</text><text x="%d" y="15" fill="#fff">%s</text></g></g></svg>`,
		totalW, label, value, label, value,
		totalW, rx,
		labelW,
		labelW, valueW,
		totalW,
		labelX, label,
		valueX, value,
	))
}
