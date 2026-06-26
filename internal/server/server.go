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
	"github.com/hrodrig/gghstats/internal/i18n"
	"github.com/hrodrig/gghstats/internal/metrics"
	"github.com/hrodrig/gghstats/internal/store"
	"github.com/hrodrig/gghstats/internal/sync"
	"github.com/hrodrig/gghstats/internal/version"
	"github.com/hrodrig/gghstats/web"
	"github.com/prometheus/client_golang/prometheus"
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
	// BadgePublic: when true (default), GET /api/v1/badge/* needs no auth (for README img embeds).
	BadgePublic bool
	// BadgeCacheMaxAge is Cache-Control max-age in seconds for badge SVG (default 300).
	BadgeCacheMaxAge int
	// PublicURL is optional base URL for embed snippets (e.g. https://gghstats.example.com); empty uses request Host.
	PublicURL string
	// SyncCoordinator serializes background and manual sync runs (nil disables sync API).
	SyncCoordinator *sync.Coordinator
	// MetricsRegistry, when set with metrics enabled, is used instead of a minimal registry (see NewMetricsRegistry).
	MetricsRegistry *prometheus.Registry
	// DomainMetrics refreshes store gauges on scrape when non-nil.
	DomainMetrics *metrics.Domain
	// ServerMetrics carries per-middleware metric vectors (rate limiter, whitelist, badges).
	// Created by serve.go and injected into middleware structs and badge handler.
	ServerMetrics *ServerMetrics
	// CustomCSSAbsPath, if non-empty, is the absolute path to a regular CSS file served at GET /theme/custom.css.
	CustomCSSAbsPath string
	// CustomCSSQuery is the cache-busting query for the layout link (e.g. "v=1715888123"); empty disables the link.
	CustomCSSQuery string
	// DefaultLocale is the fallback UI locale (GGHSTATS_DEFAULT_LOCALE, default en).
	DefaultLocale string
	// EnabledLocales lists UI locales offered in the language selector (GGHSTATS_ENABLED_LOCALES).
	EnabledLocales []string
	// RateLimiter, when non-nil, enables per-IP rate limiting (see GGHSTATS_RATE_LIMIT_* env vars).
	RateLimiter *RateLimiter
	// Whitelist, when non-nil, restricts access to whitelisted IPs on configured paths (see GGHSTATS_WHITELIST* env vars).
	Whitelist *Whitelist
	// HeadHTML is optional raw HTML injected just before </head> on every HTML page.
	HeadHTML template.HTML
	// ReverseProxyRules configures reverse-proxy mappings (see ReverseProxyRule).
	ReverseProxyRules []ReverseProxyRule
}

func withCacheControl(directive string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", directive)
		next.ServeHTTP(w, r)
	})
}

// New returns an http.Handler with all routes configured.
func New(cfg Config) http.Handler {
	cfg = normalizeLocaleConfig(cfg)
	i18n.MustLoad()
	// Register per-middleware metric vectors before mounting routes so badge
	// handler can be wrapped with its own latency/counter metrics.
	if !cfg.DisableMetrics {
		reg := cfg.MetricsRegistry
		if reg == nil {
			reg = newMetricsRegistry()
			cfg.MetricsRegistry = reg
		}
		cfg.ServerMetrics = initMiddlewareMetrics(reg)
	}
	mux := http.NewServeMux()
	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"dict": templateDict,
	}).ParseFS(web.TemplateFS, "templates/*.html"))
	mountStaticRoutes(mux, mustFaviconFS(), cfg.CustomCSSAbsPath)
	mountAPIRoutes(mux, cfg)
	mountHTMLRoutes(mux, cfg, tmpl)
	if len(cfg.ReverseProxyRules) > 0 {
		mountReverseProxyRoutes(mux, cfg.ReverseProxyRules)
	}
	return finalizeHandler(cfg, mux)
}

func mustFaviconFS() fs.FS {
	favFS, err := fs.Sub(assets.FaviconsFS, "favicons")
	if err != nil {
		panic("assets/favicons: " + err.Error())
	}
	return favFS
}

func faviconAssetNames() []string {
	return []string{
		"favicon.ico",
		"favicon.svg",
		"favicon-16x16.png",
		"favicon-32x32.png",
		"apple-touch-icon.png",
		"android-chrome-192x192.png",
		"android-chrome-512x512.png",
	}
}

func mountStaticRoutes(mux *http.ServeMux, favFS fs.FS, customCSSPath string) {
	const cache = "no-cache, must-revalidate"
	for _, name := range faviconAssetNames() {
		n := name
		mux.Handle("GET /static/"+n, withCacheControl(cache, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, favFS, n)
		})))
	}
	mux.Handle("GET /static/manifest.json", withCacheControl(cache, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	staticHandler := withCacheControl(cache, http.FileServer(http.FS(staticSub)))
	mux.Handle("GET /static/", http.StripPrefix("/static/", staticHandler))
	mux.Handle("GET /theme/custom.css", withCacheControl(cache, handleCustomCSSEndpoint(customCSSPath)))
}

func mountAPIRoutes(mux *http.ServeMux, cfg Config) {
	mux.HandleFunc("GET "+HealthzPath, handleHealthz)
	mux.HandleFunc("GET /api/repos", apiMiddleware(cfg.APIToken, handleAPIRepos(cfg.Store)))
	mux.HandleFunc("GET /api/v1/repos/{owner}/{repo}/traffic", apiMiddleware(cfg.APIToken, handleAPIRepoTraffic(cfg.Store)))
	if cfg.SyncCoordinator != nil && cfg.APIToken != "" {
		mux.HandleFunc("GET /api/v1/sync", apiMiddleware(cfg.APIToken, handleAPISyncStatus(cfg.SyncCoordinator)))
		mux.HandleFunc("POST /api/v1/sync", apiMiddleware(cfg.APIToken, handleAPISyncStart(cfg.SyncCoordinator)))
	}
	badgeHandler := badgeMiddleware(cfg, handleBadge(cfg, cfg.Store))
	if sm := cfg.ServerMetrics; sm != nil {
		badgeHandler = wrapBadgeWithMetrics(badgeHandler, sm.BadgeRequests, sm.BadgeDuration)
	}
	mux.HandleFunc("GET /api/v1/badge/{owner}/{repo}", badgeHandler)
}

func mountHTMLRoutes(mux *http.ServeMux, cfg Config, tmpl *template.Template) {
	mountSEORoutes(mux, cfg)
	repoHandler := handleRepoPage(cfg, cfg.Store, tmpl)
	indexHandler := handleIndex(cfg, cfg.Store, tmpl)
	mux.HandleFunc("GET /h2h", handleH2HPage(cfg, cfg.Store, tmpl))
	htmlNotFound := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/") {
			writeJSONNotFound(w)
			return
		}
		lb := bindPageLocale(r, cfg)
		writeBrutalistNotFound(w, r, tmpl, cfg, lb.T("not_found.title"), lb.T("not_found.heading"), path, lb.T("not_found.detail"))
	}
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			indexHandler(w, r)
			return
		}
		parts := strings.SplitN(strings.Trim(path, "/"), "/", 3)
		if len(parts) == 2 && parts[0] != "static" && parts[0] != "api" && parts[0] != "theme" && parts[0] != "h2h" {
			r.SetPathValue("owner", parts[0])
			r.SetPathValue("repo", parts[1])
			repoHandler(w, r)
			return
		}
		htmlNotFound(w, r)
	})
}

func finalizeHandler(cfg Config, mux *http.ServeMux) http.Handler {
	var h http.Handler = mux
	if cfg.DisableMetrics {
		h = logMiddleware(mux)
	} else {
		reg := cfg.MetricsRegistry
		if reg == nil {
			reg = newMetricsRegistry()
		}
		mux.Handle("GET "+MetricsPath, metricsScrapeHandler(reg, cfg.DomainMetrics))
		h = logMiddleware(wrapWithHTTPMetrics(reg, mux))
		// Inject per-middleware metric vectors so RateLimiter and Whitelist
		// record decisions at the point of reject/accept.
		if cfg.RateLimiter != nil && cfg.ServerMetrics != nil {
			cfg.RateLimiter.SetRateLimitMetrics(cfg.ServerMetrics.RateLimitedRequests)
		}
		if cfg.Whitelist != nil && cfg.ServerMetrics != nil {
			cfg.Whitelist.SetWhitelistMetrics(cfg.ServerMetrics.WhitelistRequests)
		}
	}
	skip := PublicMiddlewareSkip(cfg.ReverseProxyRules)
	if cfg.Whitelist != nil {
		h = cfg.Whitelist.Middleware(h, skip)
	}
	if cfg.RateLimiter != nil {
		h = cfg.RateLimiter.Middleware(h, skip)
	}
	return h
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
	localeBinder
	Title            string
	Version          string
	BootstrapVersion string // Bootstrap line (JS CDN + footer), e.g. 5.3.3
	StylesheetFile   string // main CSS under /static/ (go:embed), e.g. bootstrap.min.css
	Breadcrumbs      []breadcrumb
	Content          template.HTML
	PageID           string // index, h2h, repo, not_found — sidebar active state
	LocaleLinks      []localeLink
	JSI18n           template.JS
	// CustomStylesheetURL is set when GGHSTATS_CUSTOM_CSS points to a valid file (safe for href).
	CustomStylesheetURL template.URL
	// SyncUIEnabled shows the sidebar “Sync now” control (requires API token + coordinator).
	SyncUIEnabled bool
	// SyncScopeRepo when set scopes the sidebar sync to this owner/repo (repo detail pages).
	SyncScopeRepo string
	// CanonicalURL is the preferred indexing URL (no lang/sort/pagination params on index).
	CanonicalURL    template.URL
	MetaDescription string
	RobotsNoindex   bool
	// HeadHTML is optional raw HTML injected just before </head> (e.g. analytics scripts).
	HeadHTML template.HTML
}

// templateDict builds a map for {{call .Tfmt "key" (dict "a" 1)}} in templates.
func templateDict(values ...interface{}) (map[string]string, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict: expected even number of arguments")
	}
	m := make(map[string]string, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: key at index %d is not a string", i)
		}
		m[key] = fmt.Sprint(values[i+1])
	}
	return m, nil
}

func normalizeLocaleConfig(cfg Config) Config {
	if cfg.DefaultLocale == "" {
		cfg.DefaultLocale = i18n.DefaultLocale
	} else {
		cfg.DefaultLocale = i18n.NormalizeLocale(cfg.DefaultLocale)
	}
	if len(cfg.EnabledLocales) == 0 {
		cfg.EnabledLocales = []string{"en", "es", "de", "fr", "pt-br"}
	} else {
		for i, loc := range cfg.EnabledLocales {
			cfg.EnabledLocales[i] = i18n.NormalizeLocale(loc)
		}
	}
	return cfg
}

const (
	defaultPerPage          = 25
	maxPerPage              = 100
	defaultBootstrapVersion = "5.3.3"
	defaultStylesheetFile   = "bootstrap.min.css"
)

var (
	validIndexSorts = map[string]bool{
		"name": true, "stars": true, "forks": true,
		"total_views": true, "total_clones": true,
		"clones_1d": true, "clones_7d": true, "clones_30d": true,
	}
	validIndexDirs = map[string]bool{
		"asc": true, "desc": true,
	}
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

func parseIndexQueryParams(r *http.Request) (sort, dir, query string, page, perPage int) {
	sort = r.URL.Query().Get("sort")
	if sort == "" {
		sort = "total_clones"
	}
	query = strings.TrimSpace(r.URL.Query().Get("q"))
	dir = r.URL.Query().Get("dir")
	if dir == "" {
		dir = "desc"
	}
	page = parsePositiveInt(r.URL.Query().Get("page"), 1)
	perPage = parsePositiveInt(r.URL.Query().Get("per_page"), defaultPerPage)
	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	// Whitelist: reject unknown sort/dir values (defence in depth; store also validates).
	if !validIndexSorts[sort] {
		sort = "total_clones"
	}
	if !validIndexDirs[dir] {
		dir = "desc"
	}
	return sort, dir, query, page, perPage
}

func loadFilteredIndexRepos(db *store.Store, sort, dir, query string) ([]store.RepoSummary, error) {
	repos, err := db.ListRepos(sort, dir)
	if err != nil {
		return nil, err
	}
	if query != "" {
		repos = filterReposByName(repos, query)
	}
	return repos, nil
}

func repoNamesFromSummaries(repos []store.RepoSummary) []string {
	names := make([]string, len(repos))
	for i := range repos {
		names[i] = repos[i].Name
	}
	return names
}

func sumIndexKPIs(repos []store.RepoSummary) (stars, forks, clones, views int) {
	for _, rp := range repos {
		stars += rp.Stars
		forks += rp.Forks
		clones += rp.TotalClones
		views += rp.TotalViews
	}
	return stars, forks, clones, views
}

func indexReposPageSlice(repos []store.RepoSummary, page, perPage int) (start, end int, pageSlice []store.RepoSummary) {
	total := len(repos)
	start = (page - 1) * perPage
	if start > total {
		start = total
	}
	end = start + perPage
	if end > total {
		end = total
	}
	return start, end, repos[start:end]
}

func indexTotalPages(total, perPage int) int {
	if total <= 0 {
		return 1
	}
	return (total + perPage - 1) / perPage
}

func clampIndexPage(page, totalPages int) int {
	if page > totalPages {
		return totalPages
	}
	return page
}

type indexTemplatePayload struct {
	localeBinder
	ShowingLine        string
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
	SortClones1dURL    string
	SortClones7dURL    string
	SortClones30dURL   string
	SortViewsURL       string
	ListClonesAggJSON  template.JS
	ListClonesAggCount int
}

func buildIndexTemplatePayload(
	reposPage []store.RepoSummary,
	sort, dir, query string,
	page, perPage, total, start, end, totalPages int,
	kpiStars, kpiForks, kpiClones, kpiViews int,
	listClonesAggJSON template.JS,
	listClonesAggCount int,
) indexTemplatePayload {
	data := indexTemplatePayload{
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
		SortClones1dURL:    buildSortURL("clones_1d", sort, dir, query, perPage),
		SortClones7dURL:    buildSortURL("clones_7d", sort, dir, query, perPage),
		SortClones30dURL:   buildSortURL("clones_30d", sort, dir, query, perPage),
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
	return data
}

func handleIndex(cfg Config, db *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			lb := bindPageLocale(r, cfg)
			writeBrutalistNotFound(w, r, tmpl, cfg, lb.T("not_found.title"), lb.T("not_found.heading"), r.URL.Path, lb.T("not_found.detail"))
			return
		}

		sort, dir, query, page, perPage := parseIndexQueryParams(r)
		repos, err := loadFilteredIndexRepos(db, sort, dir, query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		listClonesAggCount, listClonesAggJSON, err := buildIndexListClonesChartPayload(db, repoNamesFromSummaries(repos))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		kpiStars, kpiForks, kpiClones, kpiViews := sumIndexKPIs(repos)
		total := len(repos)
		start, end, reposPage := indexReposPageSlice(repos, page, perPage)
		totalPages := indexTotalPages(total, perPage)
		page = clampIndexPage(page, totalPages)

		lb := bindPageLocale(r, cfg)
		data := buildIndexTemplatePayload(
			reposPage, sort, dir, query, page, perPage, total, start, end, totalPages,
			kpiStars, kpiForks, kpiClones, kpiViews, listClonesAggJSON, listClonesAggCount,
		)
		data.localeBinder = lb
		data.ShowingLine = lb.Tfmt("index.showing", map[string]string{
			"from":  strconv.Itoa(data.From),
			"to":    strconv.Itoa(data.To),
			"total": strconv.Itoa(data.Total),
		})

		content := executeTemplate(tmpl, "index", data)
		renderLayout(w, r, tmpl, cfg, layoutData{
			Title:   lb.T("index.title"),
			PageID:  "index",
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

func handleRepoPage(cfg Config, db *store.Store, tmpl *template.Template) http.HandlerFunc {
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
			lb := bindPageLocale(r, cfg)
			writeBrutalistNotFound(w, r, tmpl, cfg, lb.T("not_found.repo_title"), lb.T("not_found.repo_heading"), fullPath, lb.T("not_found.repo_detail"))
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

		lb := bindPageLocale(r, cfg)
		data := struct {
			localeBinder
			Repo             *store.RepoSummary
			BadgeBaseURL     string
			ViewsJSON        template.JS
			ClonesJSON       template.JS
			StarsJSON        template.JS
			Referrers        []store.PopularItem
			Paths            []store.PopularItem
			ChartClonesTitle string
			ChartViewsTitle  string
			ChartStarsTitle  string
			SyncRepoAria     string
		}{
			localeBinder:     lb,
			Repo:             summary,
			BadgeBaseURL:     publicBaseURL(r, cfg.PublicURL),
			ViewsJSON:        template.JS(viewsJSON),
			ClonesJSON:       template.JS(clonesJSON),
			StarsJSON:        template.JS(starsJSON),
			Referrers:        referrers,
			Paths:            paths,
			ChartClonesTitle: lb.Tfmt("repo.chart_clones", map[string]string{"repo": fullName}),
			ChartViewsTitle:  lb.Tfmt("repo.chart_views", map[string]string{"repo": fullName}),
			ChartStarsTitle:  lb.Tfmt("repo.chart_stars", map[string]string{"repo": fullName}),
			SyncRepoAria:     lb.Tfmt("common.sync_repo_aria", map[string]string{"repo": fullName}),
		}

		content := executeTemplate(tmpl, "repo", data)
		ld := layoutData{
			Title:       fullName,
			PageID:      "repo",
			Version:     version.Version,
			Breadcrumbs: []breadcrumb{{Label: lb.T("nav.repositories"), URL: "/"}, {Label: fullName, URL: ""}},
			Content:     content,
		}
		if cfg.SyncCoordinator != nil && cfg.APIToken != "" {
			ld.SyncScopeRepo = fullName
		}
		renderLayout(w, r, tmpl, cfg, ld)
	}
}

type notFoundContentData struct {
	localeBinder
	Heading string
	Path    string
	Detail  string
}

func writeJSONNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, `{"error":"not_found"}`)
}

func writeBrutalistNotFound(w http.ResponseWriter, r *http.Request, tmpl *template.Template, cfg Config, layoutTitle, heading, path, detail string) {
	lb := bindPageLocale(r, cfg)
	content := executeTemplate(tmpl, "not_found", notFoundContentData{
		localeBinder: lb,
		Heading:      heading,
		Path:         path,
		Detail:       detail,
	})
	renderLayoutStatus(w, r, tmpl, cfg, layoutData{
		Title:       layoutTitle,
		PageID:      "not_found",
		Version:     version.Version,
		Breadcrumbs: []breadcrumb{{Label: layoutTitle, URL: ""}},
		Content:     content,
	}, http.StatusNotFound)
}

func renderLayout(w http.ResponseWriter, r *http.Request, tmpl *template.Template, cfg Config, data layoutData) {
	renderLayoutStatus(w, r, tmpl, cfg, data, http.StatusOK)
}

func renderLayoutStatus(w http.ResponseWriter, r *http.Request, tmpl *template.Template, cfg Config, data layoutData, status int) {
	maybeSetLocaleCookie(w, r, cfg)
	data = mergeLayoutLocale(r, cfg, data)
	if cfg.CustomCSSQuery != "" {
		data.CustomStylesheetURL = template.URL("/theme/custom.css?" + cfg.CustomCSSQuery)
	}
	if cfg.SyncCoordinator != nil && cfg.APIToken != "" {
		data.SyncUIEnabled = true
	}
	data = fillLayoutDefaults(data)
	data = applyLayoutSEO(r, cfg, data)
	data.HeadHTML = cfg.HeadHTML
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
