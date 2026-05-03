package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hrodrig/gghstats/assets"
	"github.com/hrodrig/gghstats/internal/store"
	"github.com/hrodrig/gghstats/internal/version"
	"github.com/hrodrig/gghstats/web"
)

// HealthzPath is the Kubernetes-style liveness/readiness probe path (public, no auth).
const HealthzPath = "/api/v1/healthz"

// indexCloneChartMaxDays limits how many calendar days of clone points we load for the index chart
// (full filtered repo list can span years of GitHub traffic data).
const indexCloneChartMaxDays = 120

// Config holds server configuration.
type Config struct {
	Store          *store.Store
	APIToken       string // if empty, API is disabled
	DisableMetrics bool   // if true, omit /metrics and Prometheus HTTP metrics (see GGHSTATS_METRICS)
}

func withCacheControl(directive string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", directive)
		next.ServeHTTP(w, r)
	})
}

// New returns an http.Handler with all routes configured.
func New(cfg Config) http.Handler {
	mux := http.NewServeMux()

	tmpl := template.Must(template.ParseFS(web.TemplateFS, "templates/*.html"))

	favFS, err := fs.Sub(assets.FaviconsFS, "favicons")
	if err != nil {
		panic("assets/favicons: " + err.Error())
	}
	const favCache = "no-cache, must-revalidate"
	for _, name := range []string{
		"favicon.ico",
		"favicon.svg",
		"favicon-16x16.png",
		"favicon-32x32.png",
		"apple-touch-icon.png",
		"android-chrome-192x192.png",
		"android-chrome-512x512.png",
	} {
		n := name
		mux.Handle("GET /static/"+n, withCacheControl(favCache, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, favFS, n)
		})))
	}
	mux.Handle("GET /static/manifest.json", withCacheControl(favCache, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := fs.ReadFile(favFS, "manifest.json")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
		if _, err := w.Write(b); err != nil {
			slog.Error("write manifest.json", "error", err)
		}
	})))

	staticSub, _ := fs.Sub(web.StaticFS, "static")
	staticHandler := withCacheControl("no-cache, must-revalidate", http.FileServer(http.FS(staticSub)))
	mux.Handle("GET /static/", http.StripPrefix("/static/", staticHandler))

	mux.HandleFunc("GET "+HealthzPath, handleHealthz)
	mux.HandleFunc("GET /api/repos", apiMiddleware(cfg.APIToken, handleAPIRepos(cfg.Store)))

	repoHandler := handleRepoPage(cfg.Store, tmpl)
	indexHandler := handleIndex(cfg.Store, tmpl)
	htmlNotFound := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/") {
			writeJSONNotFound(w)
			return
		}
		writeBrutalistNotFound(w, tmpl, "Not found", "Page not found", path,
			"The requested page does not exist, or the URL may be incorrect.")
	}
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			indexHandler(w, r)
			return
		}
		// Route /{owner}/{repo} — must have exactly two segments
		parts := strings.SplitN(strings.Trim(path, "/"), "/", 3)
		if len(parts) == 2 && parts[0] != "static" && parts[0] != "api" {
			r.SetPathValue("owner", parts[0])
			r.SetPathValue("repo", parts[1])
			repoHandler(w, r)
			return
		}
		htmlNotFound(w, r)
	})

	if cfg.DisableMetrics {
		return logMiddleware(mux)
	}
	reg := newMetricsRegistry()
	mux.Handle("GET "+MetricsPath, metricsExporter(reg))
	return logMiddleware(wrapWithHTTPMetrics(reg, mux))
}

// --- Middleware ---

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if r.URL.Path != HealthzPath {
			slog.Info("http", "method", r.Method, "path", r.URL.Path, "dur", time.Since(start).Round(time.Millisecond))
		}
	})
}

func apiMiddleware(token string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("x-api-token") != token {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// --- Handlers ---

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok"}`)
}

func handleAPIRepos(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repos, err := db.ListRepos("total_views", "desc")
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
			return
		}

		totalStars, totalForks, totalViews, totalClones := 0, 0, 0, 0
		for _, repo := range repos {
			totalStars += repo.Stars
			totalForks += repo.Forks
			totalViews += repo.TotalViews
			totalClones += repo.TotalClones
		}

		resp := map[string]interface{}{
			"total_count":  len(repos),
			"total_stars":  totalStars,
			"total_forks":  totalForks,
			"total_views":  totalViews,
			"total_clones": totalClones,
			"items":        repos,
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(resp)
	}
}

type breadcrumb struct {
	Label string
	URL   string
}

type layoutData struct {
	Title            string
	Version          string
	BootstrapVersion string // Bootstrap line (JS CDN + footer), e.g. 5.3.3
	StylesheetFile   string // main CSS under /static/ (go:embed), e.g. bootstrap.min.css
	Breadcrumbs      []breadcrumb
	Content          template.HTML
}

const (
	defaultPerPage          = 25
	maxPerPage              = 100
	defaultBootstrapVersion = "5.3.3"
	defaultStylesheetFile   = "bootstrap.min.css"
)

func fillLayoutDefaults(d layoutData) layoutData {
	if d.BootstrapVersion == "" {
		d.BootstrapVersion = defaultBootstrapVersion
	}
	if d.StylesheetFile == "" {
		d.StylesheetFile = defaultStylesheetFile
	}
	return d
}

// buildIndexListClonesChartPayload returns JSON (for Chart.js) of daily clone totals across repoNames,
// scoped to the last indexCloneChartMaxDays ending at the newest clone date in the DB.
func buildIndexListClonesChartPayload(db *store.Store, repoNames []string) (aggCount int, js template.JS, err error) {
	js = template.JS("[]")
	if len(repoNames) == 0 {
		return 0, js, nil
	}
	minD, maxD, ok, err := db.CloneDateExtentForRepos(repoNames)
	if err != nil {
		return 0, js, err
	}
	if !ok {
		return 0, js, nil
	}
	from := minD
	if tMin, e1 := time.Parse("2006-01-02", minD); e1 == nil {
		if tMax, e2 := time.Parse("2006-01-02", maxD); e2 == nil {
			winStart := tMax.AddDate(0, 0, -indexCloneChartMaxDays)
			if tMin.Before(winStart) {
				from = winStart.Format("2006-01-02")
			}
		}
	}
	rows, err := db.AggregatedClonesByDayForRepos(repoNames, from, maxD)
	if err != nil {
		return 0, js, err
	}
	if len(rows) == 0 {
		return 0, js, nil
	}
	b, err := json.Marshal(rows)
	if err != nil {
		return 0, js, err
	}
	return len(rows), template.JS(b), nil
}

func handleIndex(db *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			writeBrutalistNotFound(w, tmpl, "Not found", "Page not found", r.URL.Path,
				"The requested page does not exist, or the URL may be incorrect.")
			return
		}

		sort := r.URL.Query().Get("sort")
		if sort == "" {
			sort = "total_views"
		}
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		dir := r.URL.Query().Get("dir")
		if dir == "" {
			dir = "desc"
		}
		page := parsePositiveInt(r.URL.Query().Get("page"), 1)
		perPage := parsePositiveInt(r.URL.Query().Get("per_page"), defaultPerPage)
		if perPage > maxPerPage {
			perPage = maxPerPage
		}

		repos, err := db.ListRepos(sort, dir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if query != "" {
			repos = filterReposByName(repos, query)
		}
		repoNames := make([]string, len(repos))
		for i := range repos {
			repoNames[i] = repos[i].Name
		}
		listClonesAggCount, listClonesAggJSON, err := buildIndexListClonesChartPayload(db, repoNames)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var kpiStars, kpiForks, kpiClones, kpiViews int
		for _, r := range repos {
			kpiStars += r.Stars
			kpiForks += r.Forks
			kpiClones += r.TotalClones
			kpiViews += r.TotalViews
		}
		total := len(repos)
		start := (page - 1) * perPage
		if start > total {
			start = total
		}
		end := start + perPage
		if end > total {
			end = total
		}
		reposPage := repos[start:end]
		totalPages := 1
		if total > 0 {
			totalPages = (total + perPage - 1) / perPage
		}
		if page > totalPages {
			page = totalPages
		}

		data := struct {
			Repos              []store.RepoSummary
			Sort               string
			Dir                string
			Query              string
			Page               int
			PerPage            int
			Total              int
			From               int
			To                 int
			KPIStars           int
			KPIForks           int
			KPIClones          int
			KPIViews           int
			PrevURL            string
			NextURL            string
			SortNameURL        string
			SortStarsURL       string
			SortForksURL       string
			SortClonesURL      string
			SortViewsURL       string
			ListClonesAggJSON  template.JS
			ListClonesAggCount int
		}{
			Repos:              reposPage,
			Sort:               sort,
			Dir:                dir,
			Query:              query,
			Page:               page,
			PerPage:            perPage,
			Total:              total,
			From:               start + 1,
			To:                 end,
			KPIStars:           kpiStars,
			KPIForks:           kpiForks,
			KPIClones:          kpiClones,
			KPIViews:           kpiViews,
			PrevURL:            buildIndexURL(sort, dir, query, page-1, perPage),
			NextURL:            buildIndexURL(sort, dir, query, page+1, perPage),
			SortNameURL:        buildSortURL("name", sort, dir, query, perPage),
			SortStarsURL:       buildSortURL("stars", sort, dir, query, perPage),
			SortForksURL:       buildSortURL("forks", sort, dir, query, perPage),
			SortClonesURL:      buildSortURL("total_clones", sort, dir, query, perPage),
			SortViewsURL:       buildSortURL("total_views", sort, dir, query, perPage),
			ListClonesAggJSON:  listClonesAggJSON,
			ListClonesAggCount: listClonesAggCount,
		}
		if total == 0 {
			data.From = 0
		}
		if page <= 1 {
			data.PrevURL = ""
		}
		if page >= totalPages {
			data.NextURL = ""
		}

		content := executeTemplate(tmpl, "index", data)
		renderLayout(w, tmpl, layoutData{
			Title:   "Repositories",
			Version: version.Version,
			Content: content,
		})
	}
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func buildSortURL(targetSort, currentSort, currentDir, query string, perPage int) string {
	nextDir := "desc"
	if currentSort == targetSort && currentDir == "desc" {
		nextDir = "asc"
	}
	return buildIndexURL(targetSort, nextDir, query, 1, perPage)
}

func buildIndexURL(sort, dir, query string, page, perPage int) string {
	if page < 1 {
		page = 1
	}
	q := url.Values{}
	q.Set("sort", sort)
	q.Set("dir", dir)
	if query != "" {
		q.Set("q", query)
	}
	q.Set("page", strconv.Itoa(page))
	q.Set("per_page", strconv.Itoa(perPage))
	return "/?" + q.Encode()
}

func filterReposByName(repos []store.RepoSummary, query string) []store.RepoSummary {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return repos
	}
	result := make([]store.RepoSummary, 0, len(repos))
	for _, repo := range repos {
		if strings.Contains(strings.ToLower(repo.Name), query) {
			result = append(result, repo)
		}
	}
	return result
}

func handleRepoPage(db *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner := r.PathValue("owner")
		repo := r.PathValue("repo")
		fullName := owner + "/" + repo

		summary, err := db.RepoByName(fullName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if summary == nil {
			fullPath := "/" + owner + "/" + repo
			writeBrutalistNotFound(w, tmpl, "Repository not found", "Repository not found", fullPath,
				"This repository is not in the database. Check the name or run a sync so it is collected.")
			return
		}

		from := time.Now().UTC().AddDate(0, 0, -365).Format("2006-01-02")
		to := time.Now().UTC().Format("2006-01-02")

		views, _ := db.ViewsByRange(fullName, from, to)
		clones, _ := db.ClonesByRange(fullName, from, to)
		stars, _ := db.StarsByRepo(fullName)
		referrers, _ := db.PopularReferrers(fullName, 14)
		paths, _ := db.PopularPaths(fullName, 14)

		viewsJSON, _ := json.Marshal(views)
		clonesJSON, _ := json.Marshal(clones)
		starsJSON, _ := json.Marshal(stars)

		data := struct {
			Repo       *store.RepoSummary
			ViewsJSON  template.JS
			ClonesJSON template.JS
			StarsJSON  template.JS
			Referrers  []store.PopularItem
			Paths      []store.PopularItem
		}{
			Repo:       summary,
			ViewsJSON:  template.JS(viewsJSON),
			ClonesJSON: template.JS(clonesJSON),
			StarsJSON:  template.JS(starsJSON),
			Referrers:  referrers,
			Paths:      paths,
		}

		content := executeTemplate(tmpl, "repo", data)
		renderLayout(w, tmpl, layoutData{
			Title:       fullName,
			Version:     version.Version,
			Breadcrumbs: []breadcrumb{{Label: fullName, URL: ""}},
			Content:     content,
		})
	}
}

type notFoundContentData struct {
	Heading string
	Path    string
	Detail  string
}

func writeJSONNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, `{"error":"not_found"}`)
}

func writeBrutalistNotFound(w http.ResponseWriter, tmpl *template.Template, layoutTitle, heading, path, detail string) {
	content := executeTemplate(tmpl, "not_found", notFoundContentData{
		Heading: heading,
		Path:    path,
		Detail:  detail,
	})
	renderLayoutStatus(w, tmpl, layoutData{
		Title:       layoutTitle,
		Version:     version.Version,
		Breadcrumbs: []breadcrumb{{Label: layoutTitle, URL: ""}},
		Content:     content,
	}, http.StatusNotFound)
}

func renderLayout(w http.ResponseWriter, tmpl *template.Template, data layoutData) {
	renderLayoutStatus(w, tmpl, data, http.StatusOK)
}

func renderLayoutStatus(w http.ResponseWriter, tmpl *template.Template, data layoutData, status int) {
	data = fillLayoutDefaults(data)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		slog.Error("render layout", "error", err)
	}
}

func executeTemplate(tmpl *template.Template, name string, data interface{}) template.HTML {
	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		slog.Error("render template", "name", name, "error", err)
		return template.HTML("<p>Error rendering page</p>")
	}
	return template.HTML(buf.String())
}
