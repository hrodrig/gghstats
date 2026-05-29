package server

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

const (
	seoRobotsContentType  = "text/plain; charset=utf-8"
	seoSitemapContentType = "application/xml; charset=utf-8"
	seoCacheControl       = "public, max-age=3600"
)

// sitemapURLCap limits repo URLs per sitemap (spec allows 50k; keeps responses bounded).
const sitemapURLCap = 10000

func mountSEORoutes(mux *http.ServeMux, cfg Config) {
	mux.HandleFunc("GET /robots.txt", handleRobots(cfg))
	mux.HandleFunc("GET /sitemap.xml", handleSitemap(cfg))
}

func seoIndexable(r *http.Request, configuredPublicURL string) bool {
	if strings.TrimSpace(configuredPublicURL) != "" {
		return true
	}
	host := strings.ToLower(requestHostName(r.Host))
	switch host {
	case "", "localhost", "127.0.0.1", "::1", "[::1]":
		return false
	}
	if strings.HasPrefix(host, "127.") {
		return false
	}
	return true
}

func requestHostName(hostport string) string {
	hostport = strings.TrimSpace(hostport)
	if hostport == "" {
		return ""
	}
	if strings.HasPrefix(hostport, "[") {
		if i := strings.LastIndex(hostport, "]"); i >= 0 {
			rest := hostport[i+1:]
			if strings.HasPrefix(rest, ":") {
				return hostport[:i+1]
			}
			return hostport
		}
		return hostport
	}
	if h, _, ok := strings.Cut(hostport, ":"); ok {
		return h
	}
	return hostport
}

func handleRobots(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", seoCacheControl)
		w.Header().Set("Content-Type", seoRobotsContentType)
		if !seoIndexable(r, cfg.PublicURL) {
			_, _ = fmt.Fprint(w, "User-agent: *\nDisallow: /\n")
			return
		}
		base := publicBaseURL(r, cfg.PublicURL)
		_, _ = fmt.Fprintf(w, "User-agent: *\nAllow: /\nDisallow: /api/\nDisallow: /metrics\nSitemap: %s/sitemap.xml\n", base)
	}
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc string `xml:"loc"`
}

func handleSitemap(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", seoCacheControl)
		w.Header().Set("Content-Type", seoSitemapContentType)
		if !seoIndexable(r, cfg.PublicURL) {
			writeEmptySitemap(w)
			return
		}
		base := publicBaseURL(r, cfg.PublicURL)
		urls := []sitemapURL{
			{Loc: base + "/"},
			{Loc: base + "/h2h"},
		}
		if cfg.Store != nil {
			repos, err := cfg.Store.ListRepos("name", "asc")
			if err != nil {
				slog.Warn("sitemap: list repos", "error", err)
			} else {
				for i, repo := range repos {
					if i >= sitemapURLCap {
						break
					}
					loc, ok := repoPageLoc(base, repo.Name)
					if !ok {
						continue
					}
					urls = append(urls, sitemapURL{Loc: loc})
				}
			}
		}
		set := sitemapURLSet{
			Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
			URLs:  urls,
		}
		w.Write([]byte(xml.Header))
		enc := xml.NewEncoder(w)
		enc.Indent("", "  ")
		if err := enc.Encode(set); err != nil {
			slog.Error("sitemap: encode", "error", err)
		}
	}
}

func writeEmptySitemap(w http.ResponseWriter) {
	set := sitemapURLSet{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_ = enc.Encode(set)
}

// repoPageLoc builds the public repo page URL (matches /{{.Name}} in templates).
func repoPageLoc(base, fullName string) (string, bool) {
	fullName = strings.TrimSpace(fullName)
	if fullName == "" || strings.Contains(fullName, " ") {
		return "", false
	}
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}
	if strings.Contains(parts[0], "/") || strings.Contains(parts[1], "/") {
		return "", false
	}
	return base + "/" + parts[0] + "/" + parts[1], true
}
