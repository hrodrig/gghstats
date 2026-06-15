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
// or whitelist checks (metrics scrape, probes, README badge embeds).
func PublicMiddlewareSkip() MiddlewareSkip {
	return MiddlewareSkip{
		Exact:    []string{MetricsPath, HealthzPath},
		Prefixes: []string{BadgePathPrefix},
	}
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
