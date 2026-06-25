package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// ReverseProxyRule defines a single reverse-proxy mapping.
type ReverseProxyRule struct {
	// Local is the local path prefix that triggers the proxy (e.g. "/kiko").
	Local string `json:"local"`
	// URL is the remote backend base URL (e.g. "https://events.example.com").
	URL string `json:"url"`
	// Headers are additional headers injected into each proxied request
	// (e.g. {"Host": "kiko-backend"}).
	Headers map[string]string `json:"headers,omitempty"`
}

// ParseReverseProxyRules parses a JSON array of ReverseProxyRule from raw.
// Returns nil when raw is empty.
func ParseReverseProxyRules(raw string) []ReverseProxyRule {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var rules []ReverseProxyRule
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		slog.Warn("invalid GGHSTATS_REVERSE_PROXY_RULES JSON", "error", err)
		return nil
	}
	return rules
}

// mountReverseProxyRoutes registers reverse-proxy handlers for each rule.
func mountReverseProxyRoutes(mux *http.ServeMux, rules []ReverseProxyRule) {
	for _, rule := range rules {
		if rule.URL == "" || rule.Local == "" {
			continue
		}
		handler := newReverseProxyHandler(rule)
		// Register for GET and POST (most common for analytics). Go 1.22+
		// ServeMux needs explicit methods to avoid conflict with "GET /".
		mux.Handle("GET "+rule.Local+"/", handler)
		mux.Handle("POST "+rule.Local+"/", handler)
		slog.Info("reverse proxy mounted",
			"local", rule.Local,
			"backend", rule.URL,
			"headers", fmt.Sprintf("%v", mapKeys(rule.Headers)),
		)
	}
}

func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// newReverseProxyHandler creates an http.Handler that reverse-proxies requests
// matching the given rule, injecting extra headers.
func newReverseProxyHandler(rule ReverseProxyRule) http.Handler {
	backendURL := mustParseURL(rule.URL)
	prefix := rule.Local

	proxy := httputil.NewSingleHostReverseProxy(backendURL)

	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = backendURL.Scheme
		req.URL.Host = backendURL.Host
		req.Host = backendURL.Host

		// Strip the local prefix so the backend receives the original path.
		// e.g. /kiko/kiko.js  →  /kiko.js, /kiko/api → /api
		trimmed := strings.TrimPrefix(req.URL.Path, prefix)
		if !strings.HasPrefix(trimmed, "/") {
			trimmed = "/" + trimmed
		}
		req.URL.Path = trimmed
		if req.URL.RawPath != "" {
			req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, prefix)
			if !strings.HasPrefix(req.URL.RawPath, "/") {
				req.URL.RawPath = "/" + req.URL.RawPath
			}
		}

		// Inject extra headers.
		for k, v := range rule.Headers {
			req.Header.Set(k, v)
		}

		slog.Debug("reverse proxy request",
			"method", req.Method,
			"from", req.URL.Path,
			"to", trimmed,
			"backend", backendURL.Host,
		)
	}

	// Override the outgoing Host header so extra headers like "Host=kiko-backend"
	// take effect (http.Transport uses req.Host, not req.Header["Host"]).
	if rule.Headers != nil {
		proxy.Transport = &headerOverrideTransport{
			extra: rule.Headers,
			base:  http.DefaultTransport,
		}
	}

	// Hide the backend's CSP from the response and fix MIME types.
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Content-Security-Policy")
		if strings.HasSuffix(resp.Request.URL.Path, ".js") {
			resp.Header.Set("Content-Type", "application/javascript; charset=utf-8")
		}
		return nil
	}

	return proxy
}

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(fmt.Sprintf("invalid reverse-proxy URL %q: %v", raw, err))
	}
	return u
}

// headerOverrideTransport wraps http.DefaultTransport and overrides the
// outgoing Host header when present in extra headers.
type headerOverrideTransport struct {
	extra map[string]string
	base  http.RoundTripper
}

func (t *headerOverrideTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if v, ok := t.extra["Host"]; ok {
		req.Host = v
	}
	return t.base.RoundTrip(req)
}
