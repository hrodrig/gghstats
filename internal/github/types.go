package github

import "time"

// RepoParent is the immediate parent of a fork (GitHub "forked from" target).
type RepoParent struct {
	FullName string `json:"full_name"`
}

// Repo represents a GitHub repository from the REST API.
type Repo struct {
	ID              int64       `json:"id"`
	FullName        string      `json:"full_name"`
	Description     *string     `json:"description"`
	StargazersCount int         `json:"stargazers_count"`
	ForksCount      int         `json:"forks_count"`
	WatchersCount   int         `json:"watchers_count"`
	OpenIssuesCount int         `json:"open_issues_count"`
	Fork            bool        `json:"fork"`
	Archived        bool        `json:"archived"`
	Parent          *RepoParent `json:"parent"`
}

// DescriptionOrEmpty returns the description or "" if nil.
func (r Repo) DescriptionOrEmpty() string {
	if r.Description == nil {
		return ""
	}
	return *r.Description
}

// ParentFullName returns the parent repo full name for forks, or "".
func (r Repo) ParentFullName() string {
	if r.Parent == nil {
		return ""
	}
	return r.Parent.FullName
}

// PullRequest represents an open PR (used to separate issues from PRs).
type PullRequest struct {
	ID int64 `json:"id"`
}

// TrafficViews represents the response from /repos/{owner}/{repo}/traffic/views.
type TrafficViews struct {
	Count   int         `json:"count"`
	Uniques int         `json:"uniques"`
	Views   []DailyStat `json:"views"`
}

// TrafficClones represents the response from /repos/{owner}/{repo}/traffic/clones.
type TrafficClones struct {
	Count   int         `json:"count"`
	Uniques int         `json:"uniques"`
	Clones  []DailyStat `json:"clones"`
}

// DailyStat holds a single day's count + uniques.
type DailyStat struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int       `json:"count"`
	Uniques   int       `json:"uniques"`
}

// Referrer represents a traffic referrer entry.
type Referrer struct {
	Referrer string `json:"referrer"`
	Count    int    `json:"count"`
	Uniques  int    `json:"uniques"`
}

// PopularPath represents a popular content path.
type PopularPath struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	Count   int    `json:"count"`
	Uniques int    `json:"uniques"`
}

// Star represents a single stargazer event with timestamp.
type Star struct {
	StarredAt time.Time `json:"starred_at"`
}
