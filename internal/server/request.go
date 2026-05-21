package server

import (
	"net/http"
	"strings"
)

// requestScheme returns "https" or "http" from TLS or X-Forwarded-Proto (first hop).
func requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return strings.ToLower(strings.TrimSpace(strings.Split(proto, ",")[0]))
	}
	return "http"
}

func requestIsHTTPS(r *http.Request) bool {
	return requestScheme(r) == "https"
}
