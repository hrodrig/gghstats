package server

import (
	"net/http/httptest"
	"testing"
)

func TestParseIndexQueryParamsDefaults(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	sort, dir, query, page, perPage := parseIndexQueryParams(r)
	if sort != "total_clones" {
		t.Errorf("sort = %q, want total_clones", sort)
	}
	if dir != "desc" {
		t.Errorf("dir = %q, want desc", dir)
	}
	if query != "" {
		t.Errorf("query = %q, want empty", query)
	}
	if page != 1 {
		t.Errorf("page = %d, want 1", page)
	}
	if perPage != 25 {
		t.Errorf("perPage = %d, want 25", perPage)
	}
}

func TestParseIndexQueryParamsValidValues(t *testing.T) {
	r := httptest.NewRequest("GET", "/?sort=stars&dir=asc&q=test&page=3&per_page=50", nil)
	sort, dir, query, page, perPage := parseIndexQueryParams(r)
	if sort != "stars" {
		t.Errorf("sort = %q, want stars", sort)
	}
	if dir != "asc" {
		t.Errorf("dir = %q, want asc", dir)
	}
	if query != "test" {
		t.Errorf("query = %q, want test", query)
	}
	if page != 3 {
		t.Errorf("page = %d, want 3", page)
	}
	if perPage != 50 {
		t.Errorf("perPage = %d, want 50", perPage)
	}
}

func TestParseIndexQueryParamsRejectsInvalidSort(t *testing.T) {
	r := httptest.NewRequest("GET", "/?sort=clones_1000000d", nil)
	sort, _, _, _, _ := parseIndexQueryParams(r)
	if sort != "total_clones" {
		t.Errorf("sort = %q, want total_clones (fallback)", sort)
	}
}

func TestParseIndexQueryParamsRejectsInvalidDir(t *testing.T) {
	r := httptest.NewRequest("GET", "/?dir=sideways", nil)
	_, dir, _, _, _ := parseIndexQueryParams(r)
	if dir != "desc" {
		t.Errorf("dir = %q, want desc (fallback)", dir)
	}
}

func TestParseIndexQueryParamsCapsPerPage(t *testing.T) {
	r := httptest.NewRequest("GET", "/?per_page=999999999", nil)
	_, _, _, _, perPage := parseIndexQueryParams(r)
	if perPage != 100 {
		t.Errorf("perPage = %d, want 100 (capped)", perPage)
	}
}

func TestParseIndexQueryParamsNegativePage(t *testing.T) {
	r := httptest.NewRequest("GET", "/?page=-5", nil)
	_, _, _, page, _ := parseIndexQueryParams(r)
	if page != 1 {
		t.Errorf("page = %d, want 1 (fallback for negative)", page)
	}
}

func TestParseIndexQueryParamsAllSortValuesAccepted(t *testing.T) {
	valid := []string{
		"name", "stars", "forks",
		"total_views", "total_clones",
		"clones_1d", "clones_7d", "clones_30d",
	}
	for _, s := range valid {
		r := httptest.NewRequest("GET", "/?sort="+s, nil)
		sort, _, _, _, _ := parseIndexQueryParams(r)
		if sort != s {
			t.Errorf("sort %q rejected, got %q", s, sort)
		}
	}
}

func TestParseIndexQueryParamsZeroPerPage(t *testing.T) {
	r := httptest.NewRequest("GET", "/?per_page=0", nil)
	_, _, _, _, perPage := parseIndexQueryParams(r)
	if perPage != 25 {
		t.Errorf("perPage = %d, want 25 (fallback for zero)", perPage)
	}
}
