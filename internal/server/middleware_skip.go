package server

import "strings"

// BadgePathPrefix is the HTTP path prefix for README embed badges (must stay public).
const BadgePathPrefix = "/api/v1/badge/"

// MiddlewareSkip lists request paths exempt from rate limiting or IP whitelist.
// Exact matches the full path; Prefixes match path prefixes (for badge routes).
type MiddlewareSkip struct {
	Exact    []string
	Prefixes []string
}

// PublicMiddlewareSkip returns paths that must stay reachable without rate limits
// or whitelist checks: metrics scrape, probes, README badge embeds, and static assets.
//
// Reverse-proxy rule prefixes are NOT included here. Rate limiting applies to those
// paths so a public relay cannot be an unbounded DoS vector against the upstream.
// Use WhitelistMiddlewareSkip when building the IP-whitelist skip list so proxy
// embeds remain reachable under a global whitelist.
func PublicMiddlewareSkip(_ []ReverseProxyRule) MiddlewareSkip {
	return MiddlewareSkip{
		Exact:    []string{MetricsPath, HealthzPath},
		Prefixes: []string{BadgePathPrefix, "/static/"},
	}
}

// WhitelistMiddlewareSkip is PublicMiddlewareSkip plus each reverse-proxy Local
// prefix. Dashboard embeds (e.g. /kiko/) must stay reachable when a global IP
// whitelist is enabled; rate limiting still applies via PublicMiddlewareSkip.
func WhitelistMiddlewareSkip(proxyRules []ReverseProxyRule) MiddlewareSkip {
	skip := PublicMiddlewareSkip(nil)
	for _, r := range proxyRules {
		if r.Local == "" {
			continue
		}
		p := r.Local
		if !strings.HasSuffix(p, "/") {
			p += "/"
		}
		skip.Prefixes = append(skip.Prefixes, p)
	}
	return skip
}

func (s MiddlewareSkip) matches(path string) bool {
	for _, p := range s.Exact {
		if path == p {
			return true
		}
	}
	for _, p := range s.Prefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
