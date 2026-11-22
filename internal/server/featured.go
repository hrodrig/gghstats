package server

import (
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hrodrig/gghstats/internal/store"
	"github.com/hrodrig/gghstats/internal/version"
)

// validFeaturedSorts whitelists the sortable fields on the /featured page.
// "sort" is the operator display order (insertion); name and stars sort by
// upstream metadata.
var validFeaturedSorts = map[string]bool{
	"sort":  true,
	"name":  true,
	"stars": true,
}

// featuredPageData is the payload for the /featured showcase template.
type featuredPageData struct {
	localeBinder
	Featured     []store.Featured
	Sort         string
	Dir          string
	Query        string
	Page         int
	PerPage      int
	Total        int
	From         int
	To           int
	PrevURL      string
	NextURL      string
	SortOrderURL string
	SortNameURL  string
	SortStarsURL string
	ShowingLine  string
}

func handleFeaturedPage(cfg Config, db *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/featured" {
			lb := bindPageLocale(r, cfg)
			writeBrutalistNotFound(w, r, tmpl, cfg, lb.T("not_found.title"), lb.T("not_found.heading"), r.URL.Path, lb.T("not_found.detail"))
			return
		}

		sort, dir, query, page, perPage := parseFeaturedQueryParams(r)
		featured, total, err := db.FilterFeatured(query, sort, dir, page, perPage)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for i := range featured {
			featured[i].UpstreamDescription = stripEmojiShortcodes(featured[i].UpstreamDescription)
		}

		totalPages := indexTotalPages(total, perPage)
		page = clampIndexPage(page, totalPages)

		data := featuredPageData{
			Featured:     featured,
			Sort:         sort,
			Dir:          dir,
			Query:        query,
			Page:         page,
			PerPage:      perPage,
			Total:        total,
			From:         (page-1)*perPage + 1,
			To:           (page-1)*perPage + len(featured),
			SortOrderURL: buildFeaturedSortURL("sort", sort, dir, query, perPage),
			SortNameURL:  buildFeaturedSortURL("name", sort, dir, query, perPage),
			SortStarsURL: buildFeaturedSortURL("stars", sort, dir, query, perPage),
			PrevURL:      buildFeaturedURL(sort, dir, query, page-1, perPage),
			NextURL:      buildFeaturedURL(sort, dir, query, page+1, perPage),
		}
		if total == 0 {
			data.From = 0
			data.To = 0
		}
		if page <= 1 {
			data.PrevURL = ""
		}
		if page >= totalPages {
			data.NextURL = ""
		}

		lb := bindPageLocale(r, cfg)
		data.localeBinder = lb
		data.ShowingLine = lb.Tfmt("index.showing", map[string]string{
			"from":  strconv.Itoa(data.From),
			"to":    strconv.Itoa(data.To),
			"total": strconv.Itoa(data.Total),
		})

		content := executeTemplate(tmpl, "featured", data)
		renderLayout(w, r, tmpl, cfg, layoutData{
			Title:   lb.T("featured.title"),
			PageID:  "featured",
			Version: version.Version,
			Breadcrumbs: []breadcrumb{
				{Label: lb.T("nav.home"), URL: "/"},
				{Label: lb.T("featured.title"), URL: ""},
			},
			Content: content,
		})
	}
}

func parseFeaturedQueryParams(r *http.Request) (sort, dir, query string, page, perPage int) {
	sort = r.URL.Query().Get("sort")
	if sort == "" {
		sort = "sort"
	}
	if !validFeaturedSorts[sort] {
		sort = "sort"
	}
	dir = r.URL.Query().Get("dir")
	if dir == "" {
		dir = "asc"
	}
	if !validIndexDirs[dir] {
		dir = "asc"
	}
	query = strings.TrimSpace(r.URL.Query().Get("q"))
	page = parsePositiveInt(r.URL.Query().Get("page"), 1)
	perPage = parsePositiveInt(r.URL.Query().Get("per_page"), defaultPerPage)
	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	return sort, dir, query, page, perPage
}

func buildFeaturedSortURL(targetSort, currentSort, currentDir, query string, perPage int) string {
	nextDir := "asc"
	if currentSort == targetSort && currentDir == "asc" {
		nextDir = "desc"
	}
	return buildFeaturedURL(targetSort, nextDir, query, 1, perPage)
}

func buildFeaturedURL(sort, dir, query string, page, perPage int) string {
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
	return "/featured?" + q.Encode()
}
