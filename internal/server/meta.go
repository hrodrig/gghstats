package server

import (
	"html/template"
	"net/http"
	"net/url"
	"strings"
)

// applyLayoutSEO sets canonical URL, meta description, and robots for HTML layout pages.
func applyLayoutSEO(r *http.Request, cfg Config, data layoutData) layoutData {
	base := publicBaseURL(r, cfg.PublicURL)
	data.CanonicalURL = template.URL(pageCanonicalURL(base, r, data.PageID))
	data.MetaDescription = layoutMetaDescription(data, r)
	if data.PageID == "not_found" {
		data.RobotsNoindex = true
	}
	return data
}

// pageCanonicalURL returns the preferred URL for indexing (no lang/sort/page noise on index).
func pageCanonicalURL(base string, r *http.Request, pageID string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	path := r.URL.Path
	if path == "" {
		path = "/"
	}

	switch pageID {
	case "index":
		if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
			return base + "/?" + url.Values{"q": {q}}.Encode()
		}
		return base + "/"
	case "h2h":
		return h2hCanonicalURL(base, r.URL.Query())
	case "repo":
		return base + path
	default:
		return base + path
	}
}

func h2hCanonicalURL(base string, q url.Values) string {
	a := strings.TrimSpace(q.Get("a"))
	b := strings.TrimSpace(q.Get("b"))
	if a == "" || b == "" {
		return base + "/h2h"
	}
	v := url.Values{}
	v.Set("a", a)
	v.Set("b", b)
	if w := strings.TrimSpace(q.Get("w")); w != "" {
		v.Set("w", w)
	}
	return base + "/h2h?" + v.Encode()
}

func layoutMetaDescription(data layoutData, r *http.Request) string {
	switch data.PageID {
	case "index":
		return data.T("meta.index")
	case "h2h":
		a := strings.TrimSpace(r.URL.Query().Get("a"))
		b := strings.TrimSpace(r.URL.Query().Get("b"))
		if a != "" && b != "" {
			return data.Tfmt("meta.h2h_compare", map[string]string{"repoA": a, "repoB": b})
		}
		return data.T("meta.h2h")
	case "repo":
		owner := r.PathValue("owner")
		repo := r.PathValue("repo")
		if owner != "" && repo != "" {
			return data.Tfmt("meta.repo", map[string]string{"repo": owner + "/" + repo})
		}
		return data.T("meta.repo_generic")
	case "not_found":
		return data.T("meta.not_found")
	default:
		return data.T("meta.index")
	}
}
