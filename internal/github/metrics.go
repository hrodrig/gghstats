package github

import (
	"net/http"

	"github.com/hrodrig/gghstats/internal/metrics"
)

// MetricsRecorder records GitHub API calls (implemented by metrics.Domain).
type MetricsRecorder interface {
	ObserveGitHubRequest(endpoint, status string)
	SetGitHubRateLimitRemaining(remaining int)
	SetGitHubRateLimitReset(resetUnix int64)
}

// SetMetrics attaches an optional Prometheus recorder to the client.
func (c *Client) SetMetrics(rec MetricsRecorder) {
	c.metrics = rec
}

func (c *Client) recordResponse(path string, resp *http.Response, err error) {
	if resp != nil {
		if rem, ok := metrics.GitHubRateLimitRemaining(resp); ok {
			c.rateMu.Lock()
			c.lastRateLimit = rem
			c.rateMu.Unlock()
		}
	}
	if c.metrics == nil {
		return
	}
	endpoint := metrics.NormalizeGitHubEndpoint(path)
	status := metrics.ClassifyGitHubOutcome(resp, err)
	c.metrics.ObserveGitHubRequest(endpoint, status)
	if resp != nil {
		if rem, ok := metrics.GitHubRateLimitRemaining(resp); ok {
			c.metrics.SetGitHubRateLimitRemaining(rem)
		}
		if reset, ok := metrics.GitHubRateLimitReset(resp); ok {
			c.metrics.SetGitHubRateLimitReset(reset)
		}
	}
}

// LastRateLimitRemaining returns the last observed X-RateLimit-Remaining, or -1 if unknown.
func (c *Client) LastRateLimitRemaining() int {
	c.rateMu.Lock()
	defer c.rateMu.Unlock()
	return c.lastRateLimit
}
