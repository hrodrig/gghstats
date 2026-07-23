package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func apiGET(t *testing.T, h http.Handler, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("x-api-token", token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestAPIReposQueryAndPagination(t *testing.T) {
	db := testStore(t)
	_ = db.UpsertRepo("z/zebra", "", 1, 0, 1, 0, 0, false, false, "")
	_ = db.UpsertRepo("a/alpha", "", 50, 0, 50, 0, 0, false, false, "")
	_ = db.UpsertRepo("a/other", "", 2, 0, 2, 0, 0, false, false, "")

	h := New(Config{Store: db, APIToken: "tok", DisableMetrics: true})

	w := apiGET(t, h, "/api/repos?sort=name&dir=asc&q=a/", "tok")
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["total_count"].(float64) != 2 {
		t.Fatalf("filtered count = %v", body["total_count"])
	}
	items := body["items"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("items len = %d", len(items))
	}
	first := items[0].(map[string]interface{})
	if first["name"] != "a/alpha" {
		t.Fatalf("sort name asc first = %v", first["name"])
	}

	w = apiGET(t, h, "/api/repos?sort=name&dir=asc&page=1&per_page=1", "tok")
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["total_count"].(float64) != 3 {
		t.Fatalf("total_count = %v", body["total_count"])
	}
	if body["page"].(float64) != 1 || body["per_page"].(float64) != 1 || body["total_pages"].(float64) != 3 {
		t.Fatalf("pagination meta = %#v", body)
	}
	items = body["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("page size = %d", len(items))
	}
}

func TestAPIRepoDetailNotFoundAndOK(t *testing.T) {
	db := testStore(t)
	_ = db.UpsertRepo("o/r", "desc", 3, 1, 3, 0, 0, false, false, "")
	today := time.Now().UTC()
	for i := 0; i < 14; i++ {
		d := today.AddDate(0, 0, -i).Format("2006-01-02")
		_ = db.UpsertClone("o/r", d, 10, 5)
		_ = db.UpsertView("o/r", d, 20, 8)
	}

	h := New(Config{Store: db, APIToken: "tok", DisableMetrics: true})

	w := apiGET(t, h, "/api/v1/repos/missing/repo", "tok")
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing status=%d", w.Code)
	}

	w = apiGET(t, h, "/api/v1/repos/o/r", "tok")
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["repo"] == nil {
		t.Fatal("missing repo")
	}
	if _, ok := body["momentum_7d"]; !ok {
		t.Fatal("missing momentum_7d")
	}
}

func TestAPIRepoStarsAndPopular(t *testing.T) {
	db := testStore(t)
	_ = db.UpsertRepo("o/r", "", 1, 0, 1, 0, 0, false, false, "")
	_ = db.UpsertStar("o/r", "2026-01-01", 5)
	_ = db.UpsertStar("o/r", "2026-02-01", 10)

	h := New(Config{Store: db, APIToken: "tok", DisableMetrics: true})

	w := apiGET(t, h, "/api/v1/repos/o/r/stars", "tok")
	if w.Code != http.StatusOK {
		t.Fatalf("stars status=%d", w.Code)
	}
	var starsBody map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &starsBody)
	stars := starsBody["stars"].([]interface{})
	if len(stars) != 2 {
		t.Fatalf("stars len=%d", len(stars))
	}

	w = apiGET(t, h, "/api/v1/repos/o/r/popular", "tok")
	if w.Code != http.StatusOK {
		t.Fatalf("popular status=%d", w.Code)
	}
	var pop map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &pop)
	if pop["days"].(float64) != 14 {
		t.Fatalf("days=%v", pop["days"])
	}
	if pop["referrers"] == nil || pop["paths"] == nil {
		t.Fatal("missing referrers/paths")
	}

	w = apiGET(t, h, "/api/v1/repos/no/such/stars", "tok")
	if w.Code != http.StatusNotFound {
		t.Fatalf("stars 404 status=%d", w.Code)
	}
}

func TestAPIH2HValidationAndOK(t *testing.T) {
	db := seedH2HRepos(t)
	h := New(Config{Store: db, APIToken: "tok", DisableMetrics: true})

	cases := []struct {
		path string
		code int
	}{
		{"/api/v1/h2h", http.StatusBadRequest},
		{"/api/v1/h2h?a=a/one", http.StatusBadRequest},
		{"/api/v1/h2h?a=bad&b=a/one", http.StatusBadRequest},
		{"/api/v1/h2h?a=a/one&b=a/one", http.StatusBadRequest},
		{"/api/v1/h2h?a=missing/x&b=a/one", http.StatusNotFound},
	}
	for _, tc := range cases {
		w := apiGET(t, h, tc.path, "tok")
		if w.Code != tc.code {
			t.Errorf("%s status=%d want %d body=%s", tc.path, w.Code, tc.code, w.Body.String())
		}
	}

	w := apiGET(t, h, "/api/v1/h2h?a=a/one&b=b/two&w=7d", "tok")
	if w.Code != http.StatusOK {
		t.Fatalf("ok status=%d body=%s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	res := body["result"].(map[string]interface{})
	if _, ok := res["score_a"]; !ok {
		t.Fatalf("expected snake_case score_a in %#v", res)
	}
	if body["charts"] == nil {
		t.Fatal("missing charts")
	}
}

func TestAPIIndexClonesChart(t *testing.T) {
	db := seedH2HRepos(t)
	h := New(Config{Store: db, APIToken: "tok", DisableMetrics: true})
	w := apiGET(t, h, "/api/v1/charts/index-clones", "tok")
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["series"] == nil {
		t.Fatal("missing series")
	}
}

func TestAPIDogfoodUnauthorized(t *testing.T) {
	db := testStore(t)
	_ = db.UpsertRepo("o/r", "", 1, 0, 1, 0, 0, false, false, "")
	h := New(Config{Store: db, APIToken: "tok", DisableMetrics: true})

	paths := []string{
		"/api/v1/repos/o/r",
		"/api/v1/repos/o/r/stars",
		"/api/v1/repos/o/r/popular",
		"/api/v1/h2h?a=o/r&b=x/y",
		"/api/v1/charts/index-clones",
	}
	for _, p := range paths {
		w := apiGET(t, h, p, "")
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s status=%d want 401", p, w.Code)
		}
	}
}
