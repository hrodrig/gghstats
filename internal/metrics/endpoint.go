package metrics

import (
	"regexp"
	"strings"
)

var (
	reposPathRe = regexp.MustCompile(`^/repos/[^/]+/[^/]+`)
)

type repoPathRule struct {
	fragment string
	label    string
}

// repoPathRules are checked in order; more specific traffic paths before generic /repos/.
var repoPathRules = []repoPathRule{
	{"/traffic/views", "traffic_views"},
	{"/traffic/clones", "traffic_clones"},
	{"/traffic/popular/referrers", "traffic_referrers"},
	{"/traffic/popular/paths", "traffic_paths"},
	{"/stargazers", "stargazers"},
	{"/pulls", "pulls"},
}

// NormalizeGitHubEndpoint maps a request path to a low-cardinality label.
func NormalizeGitHubEndpoint(path string) string {
	if path == "" {
		return "unknown"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if path == "/user/repos" || strings.HasPrefix(path, "/user/repos?") {
		return "user_repos"
	}
	if !strings.HasPrefix(path, "/repos/") {
		return "other"
	}
	return normalizeRepoPath(path)
}

func normalizeRepoPath(path string) string {
	for _, r := range repoPathRules {
		if strings.Contains(path, r.fragment) {
			return r.label
		}
	}
	if reposPathRe.MatchString(path) {
		return "repos"
	}
	return "other"
}
