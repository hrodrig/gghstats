package server

import (
	"net/http"
	"strings"
)

// ParseCORSOrigins splits a comma-separated GGHSTATS_CORS_ORIGINS value.
// Empty or whitespace-only input yields nil (treat as "*").
func ParseCORSOrigins(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// CORSIsOpen reports whether the effective allow list is wildcard "*".
func CORSIsOpen(origins []string) bool {
	if len(origins) == 0 {
		return true
	}
	for _, o := range origins {
		if o == "*" {
			return true
		}
	}
	return false
}

// setAPICORS sets Access-Control-Allow-Origin for authenticated JSON APIs.
// Empty origins → "*". Otherwise echo Origin when it matches the allow list;
// if the request has no Origin (non-browser), use the first configured origin.
func setAPICORS(w http.ResponseWriter, r *http.Request, origins []string) {
	if CORSIsOpen(origins) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		return
	}
	reqOrigin := strings.TrimSpace(r.Header.Get("Origin"))
	if reqOrigin != "" {
		for _, o := range origins {
			if o == reqOrigin {
				w.Header().Set("Access-Control-Allow-Origin", reqOrigin)
				w.Header().Add("Vary", "Origin")
				return
			}
		}
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origins[0])
}
