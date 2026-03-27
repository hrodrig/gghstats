package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

const defaultBaseURL = "https://api.github.com"

// Client talks to the GitHub REST API for traffic endpoints.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewClient returns a Client configured with the given token.
func NewClient(token string) *Client {
	return &Client{
		BaseURL: defaultBaseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ListRepos returns all repositories accessible to the authenticated user.
// If includePrivate is true, private repos are included.
func (c *Client) ListRepos(includePrivate bool) ([]Repo, error) {
	visibility := "public"
	if includePrivate {
		visibility = "all"
	}
	var all []Repo
	if err := c.getPaginated(fmt.Sprintf("/user/repos?visibility=%s&per_page=100", visibility), &all); err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}
	return all, nil
}

// Repo returns repository metadata from GET /repos/{owner}/{repo}, including parent for forks.
func (c *Client) Repo(fullName string) (*Repo, error) {
	var out Repo
	if err := c.get(fmt.Sprintf("/repos/%s", fullName), &out); err != nil {
		return nil, fmt.Errorf("repo: %w", err)
	}
	return &out, nil
}

// OpenPullRequests returns open PRs for a repo (used to separate issues from PRs).
func (c *Client) OpenPullRequests(repo string) ([]PullRequest, error) {
	var all []PullRequest
	if err := c.getPaginated(fmt.Sprintf("/repos/%s/pulls?state=open&per_page=100", repo), &all); err != nil {
		return nil, fmt.Errorf("pull requests: %w", err)
	}
	return all, nil
}

// Views fetches /repos/{owner}/{repo}/traffic/views (last 14 days).
func (c *Client) Views(repo string) (*TrafficViews, error) {
	var out TrafficViews
	if err := c.get(fmt.Sprintf("/repos/%s/traffic/views", repo), &out); err != nil {
		return nil, fmt.Errorf("views: %w", err)
	}
	return &out, nil
}

// Clones fetches /repos/{owner}/{repo}/traffic/clones (last 14 days).
func (c *Client) Clones(repo string) (*TrafficClones, error) {
	var out TrafficClones
	if err := c.get(fmt.Sprintf("/repos/%s/traffic/clones", repo), &out); err != nil {
		return nil, fmt.Errorf("clones: %w", err)
	}
	return &out, nil
}

// Referrers fetches /repos/{owner}/{repo}/traffic/popular/referrers.
func (c *Client) Referrers(repo string) ([]Referrer, error) {
	var out []Referrer
	if err := c.get(fmt.Sprintf("/repos/%s/traffic/popular/referrers", repo), &out); err != nil {
		return nil, fmt.Errorf("referrers: %w", err)
	}
	return out, nil
}

// PopularPaths fetches /repos/{owner}/{repo}/traffic/popular/paths.
func (c *Client) PopularPaths(repo string) ([]PopularPath, error) {
	var out []PopularPath
	if err := c.get(fmt.Sprintf("/repos/%s/traffic/popular/paths", repo), &out); err != nil {
		return nil, fmt.Errorf("paths: %w", err)
	}
	return out, nil
}

// Stargazers fetches the full list of stargazers with timestamps.
// Requires the special Accept header for starred_at field.
func (c *Client) Stargazers(repo string) ([]Star, error) {
	var all []Star
	path := fmt.Sprintf("/repos/%s/stargazers?per_page=100", repo)
	if err := c.getPaginatedWithAccept(path, "application/vnd.github.v3.star+json", &all); err != nil {
		return nil, fmt.Errorf("stargazers: %w", err)
	}
	return all, nil
}

// --- HTTP helpers ---

func (c *Client) get(path string, dest interface{}) error {
	return c.getWithAccept(path, "application/vnd.github+json", dest)
}

func (c *Client) getWithAccept(path, accept string, dest interface{}) error {
	url := c.BaseURL + path

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(dest)
}

// getPaginated follows Link headers to collect all pages into a single slice.
func (c *Client) getPaginated(path string, dest interface{}) error {
	return c.getPaginatedWithAccept(path, "application/vnd.github+json", dest)
}

func (c *Client) getPaginatedWithAccept(path, accept string, dest interface{}) error {
	slicePtr, ok := dest.(*[]Star)
	_ = slicePtr
	// Use generic approach: accumulate raw JSON arrays
	var allRaw []json.RawMessage
	currentPath := path

	for currentPath != "" {
		url := c.BaseURL + currentPath

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", accept)
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}

		var page []json.RawMessage
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return err
		}
		resp.Body.Close()

		allRaw = append(allRaw, page...)

		currentPath = nextPagePath(resp.Header.Get("Link"))
	}

	combined, err := json.Marshal(allRaw)
	if err != nil {
		return err
	}

	if ok {
		return json.Unmarshal(combined, slicePtr)
	}
	return json.Unmarshal(combined, dest)
}

var linkNextRe = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

func nextPagePath(linkHeader string) string {
	if linkHeader == "" {
		return ""
	}
	matches := linkNextRe.FindStringSubmatch(linkHeader)
	if len(matches) < 2 {
		return ""
	}
	// The URL is absolute; extract the path+query portion
	fullURL := matches[1]
	// Find the path starting from /
	for i, ch := range fullURL {
		if ch == '/' && i > 8 { // skip "https://"
			return fullURL[i:]
		}
	}
	return ""
}
