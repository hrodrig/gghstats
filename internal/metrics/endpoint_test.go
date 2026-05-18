package metrics

import "testing"

func TestNormalizeGitHubEndpoint(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"", "unknown"},
		{"repos/o/r", "repos"},
		{"/user/repos", "user_repos"},
		{"/user/repos?visibility=all&per_page=100", "user_repos"},
		{"/repos/o/r", "repos"},
		{"/repos/o/r/traffic/views", "traffic_views"},
		{"/repos/o/r/traffic/clones", "traffic_clones"},
		{"/repos/o/r/traffic/popular/referrers", "traffic_referrers"},
		{"/repos/o/r/traffic/popular/paths", "traffic_paths"},
		{"/repos/o/r/stargazers?per_page=100", "stargazers"},
		{"/repos/o/r/pulls?state=open", "pulls"},
		{"/unknown", "other"},
	}
	for _, tt := range tests {
		if got := NormalizeGitHubEndpoint(tt.path); got != tt.want {
			t.Errorf("NormalizeGitHubEndpoint(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
