package server

import (
	"net"
	"net/http"
	"os"
	"strings"
)

// Whitelist controls access by client IP/CIDR, optionally scoped to specific paths.
type Whitelist struct {
	cidrs     []*net.IPNet
	paths     []string // empty = all paths
	apiToken  string   // when set, valid x-api-token bypasses the whitelist
}

// WhitelistConfig holds parsed whitelist parameters.
type WhitelistConfig struct {
	CIDRs string // comma-separated IPs/CIDRs (empty = disabled)
	Paths string // comma-separated path prefixes (empty = all paths)
}

// ParseWhitelistEnv reads GGHSTATS_WHITELIST and GGHSTATS_WHITELIST_PATHS.
func ParseWhitelistEnv() WhitelistConfig {
	return WhitelistConfig{
		CIDRs: strings.TrimSpace(os.Getenv("GGHSTATS_WHITELIST")),
		Paths: strings.TrimSpace(os.Getenv("GGHSTATS_WHITELIST_PATHS")),
	}
}

// NewWhitelist parses a WhitelistConfig and returns a ready-to-use Whitelist.
// When apiToken is non-empty, requests bearing a matching x-api-token header
// bypass the IP check (token-protected API routes remain gated by apiMiddleware).
// Returns nil if no CIDRs are configured (whitelist disabled).
func NewWhitelist(cfg WhitelistConfig, apiToken string) *Whitelist {
	if cfg.CIDRs == "" {
		return nil
	}
	w := &Whitelist{apiToken: apiToken}
	for _, raw := range strings.Split(cfg.CIDRs, ",") {
		cidr := strings.TrimSpace(raw)
		if cidr == "" {
			continue
		}
		// If it doesn't contain a slash, treat as /32 (single IP).
		if !strings.Contains(cidr, "/") {
			cidr += "/32"
		}
		_, net, err := net.ParseCIDR(cidr)
		if err != nil {
			continue // skip invalid entries
		}
		w.cidrs = append(w.cidrs, net)
	}
	if len(w.cidrs) == 0 {
		return nil
	}
	if cfg.Paths != "" {
		for _, p := range strings.Split(cfg.Paths, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				if !strings.HasPrefix(p, "/") {
					p = "/" + p
				}
				w.paths = append(w.paths, p)
			}
		}
	}
	return w
}

// Middleware returns an HTTP middleware that only allows requests from
// whitelisted IPs on configured paths. If paths are empty, all routes are
// protected. Exempt paths (metrics, healthz, badge embeds) always pass through.
func (w *Whitelist) Middleware(next http.Handler, exempt MiddlewareSkip) http.Handler {
	return http.HandlerFunc(func(wr http.ResponseWriter, r *http.Request) {
		if exempt.matches(r.URL.Path) {
			next.ServeHTTP(wr, r)
			return
		}
		if !w.pathMatches(r.URL.Path) {
			next.ServeHTTP(wr, r)
			return
		}
		if w.apiToken != "" && r.Header.Get("x-api-token") == w.apiToken {
			next.ServeHTTP(wr, r)
			return
		}
		ip := clientIP(r)
		if !w.allowed(ip) {
			wr.Header().Set("Content-Type", "application/json")
			wr.WriteHeader(http.StatusForbidden)
			_, _ = wr.Write([]byte(`{"error":"ip_not_whitelisted"}`))
			return
		}
		next.ServeHTTP(wr, r)
	})
}

func (w *Whitelist) allowed(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, cidr := range w.cidrs {
		if cidr.Contains(parsed) {
			return true
		}
	}
	return false
}

// pathMatches returns true if the request path falls under any configured path
// prefix, or if no paths are configured (protect all routes).
func (w *Whitelist) pathMatches(requestPath string) bool {
	if len(w.paths) == 0 {
		return true // protect everything
	}
	for _, prefix := range w.paths {
		// Exact match (with or without trailing slash).
		if requestPath == prefix || requestPath == strings.TrimSuffix(prefix, "/") {
			return true
		}
		// Prefix match: /api/ matches /api/repos, /api/v1/sync, etc.
		if strings.HasPrefix(requestPath, ensureTrailingSlash(prefix)) {
			return true
		}
	}
	return false
}

func ensureTrailingSlash(p string) string {
	if strings.HasSuffix(p, "/") {
		return p
	}
	return p + "/"
}
