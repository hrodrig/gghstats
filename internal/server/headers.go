package server

import (
	"net/http"
	"strings"
)

// defaultCSP is a Report-Only / enforce baseline for the embedded dashboard.
// Inline scripts and known CDN hosts match layout.html (Chart.js, Luxon, Bootstrap).
const defaultCSP = "default-src 'self'; " +
	"script-src 'self' 'unsafe-inline' https://unpkg.com https://cdn.jsdelivr.net; " +
	"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
	"font-src 'self' https://fonts.gstatic.com; " +
	"img-src 'self' data:; " +
	"connect-src 'self'; " +
	"frame-ancestors 'none'; " +
	"base-uri 'self'; " +
	"form-action 'self'"

// securityHeadersMiddleware sets a conservative baseline for browser responses.
func securityHeadersMiddleware(cfg Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		applyCSPHeaders(h, cfg)
		next.ServeHTTP(w, r)
	})
}

func applyCSPHeaders(h http.Header, cfg Config) {
	mode := strings.ToLower(strings.TrimSpace(cfg.CSPMode))
	enforce := mode == "enforce"
	if enforce && strings.TrimSpace(string(cfg.HeadHTML)) != "" {
		// Custom HeadHTML may inject arbitrary scripts; stay report-only.
		h.Set("Content-Security-Policy-Report-Only", defaultCSP)
		return
	}
	if enforce {
		h.Set("Content-Security-Policy", defaultCSP)
		return
	}
	h.Set("Content-Security-Policy-Report-Only", defaultCSP)
}
