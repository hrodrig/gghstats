package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/h2h"
	"github.com/hrodrig/gghstats/internal/i18n"
	"github.com/hrodrig/gghstats/internal/store"
)

func seedH2HRepos(t *testing.T) *store.Store {
	t.Helper()
	db := testStore(t)
	today := time.Now().UTC()
	for i := 0; i < 30; i++ {
		d := today.AddDate(0, 0, -i).Format("2006-01-02")
		_ = db.UpsertRepo("a/one", "one", 1, 0, 1, 0, 0, false, false, "")
		_ = db.UpsertRepo("b/two", "two", 1, 0, 1, 0, 0, false, false, "")
		_ = db.UpsertClone("a/one", d, 10+i, 5)
		_ = db.UpsertClone("b/two", d, 5+i, 3)
		_ = db.UpsertView("a/one", d, 8, 4)
		_ = db.UpsertView("b/two", d, 4, 2)
	}
	return db
}

func TestH2HPage_shell(t *testing.T) {
	handler := New(Config{Store: testStore(t)})
	req := httptest.NewRequest(http.MethodGet, "/h2h", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Head to Head") || !strings.Contains(body, `name="a"`) {
		t.Error("expected H2H form shell")
	}
}

func TestH2HPage_compare7d(t *testing.T) {
	db := seedH2HRepos(t)
	handler := New(Config{Store: db})
	req := httptest.NewRequest(http.MethodGet, "/h2h?a=a/one&b=b/two&w=7d", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "H2H share") || !strings.Contains(body, "gghstatsH2HChartData") {
		t.Error("expected comparison scores and chart JSON")
	}
	if !strings.Contains(body, "a/one") || !strings.Contains(body, "b/two") {
		t.Error("expected repo names in response")
	}
}

func TestH2HPage_validationErrors(t *testing.T) {
	db := testStore(t)
	_ = db.UpsertRepo("a/one", "x", 1, 0, 1, 0, 0, false, false, "")
	handler := New(Config{Store: db})

	req := httptest.NewRequest(http.MethodGet, "/h2h?a=bad&b=a/one", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), "valid owner/repo") {
		t.Error("expected invalid name error")
	}

	req = httptest.NewRequest(http.MethodGet, "/h2h?a=a/one&b=a/one", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), "two different") {
		t.Error("expected same-repo error")
	}

	req = httptest.NewRequest(http.MethodGet, "/h2h?a=a/one&b=missing/r", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), "not found in the database") {
		t.Error("expected missing repo error")
	}
}

func TestH2HHelpers(t *testing.T) {
	b := i18n.MustLoad()
	if b.IntervalLabel("en", h2h.Interval30d) != "30 days" {
		t.Error("chart span label 30d")
	}
	if b.IntervalLabel("en", h2h.IntervalTotal) != "All time" {
		t.Error("chart span label total")
	}
	if !strings.Contains(b.MomentumChartLabel("en", h2h.Interval30d), "30d") {
		t.Error("momentum label 30d")
	}
	if h2hQueryURL("a/r", "b/r", "30d") != "/h2h?a=a%2Fr&b=b%2Fr&w=30d" {
		t.Errorf("query URL = %q", h2hQueryURL("a/r", "b/r", "30d"))
	}
	if h2hQueryURL("a/r", "b/r", "7d") != "/h2h?a=a%2Fr&b=b%2Fr" {
		t.Errorf("default interval omitted: %q", h2hQueryURL("a/r", "b/r", "7d"))
	}

	now := time.Date(2026, 5, 19, 0, 0, 0, 0, time.UTC)
	chartFrom, cloneFrom := h2hChartFromDates(now, h2h.Interval7d)
	if chartFrom != "2026-05-12" || cloneFrom != "2026-04-28" {
		t.Errorf("7d from dates chart=%q clone=%q", chartFrom, cloneFrom)
	}
	totalChart, totalClone := h2hChartFromDates(now, h2h.IntervalTotal)
	if totalChart != "2000-01-01" || totalClone != totalChart {
		t.Errorf("total from dates = %q %q", totalChart, totalClone)
	}

	rows := []store.DayRow{{Date: "2026-05-01", Count: 1}, {Date: "2026-05-10", Count: 2}}
	trimmed := trimDayRowsFrom(rows, "2026-05-05")
	if len(trimmed) != 1 || trimmed[0].Date != "2026-05-10" {
		t.Fatalf("trimmed = %+v", trimmed)
	}
}

func TestBuildH2HChartJSON(t *testing.T) {
	db := seedH2HRepos(t)
	js, ok := buildH2HChartJSON(db, "a/one", "b/two", h2h.Interval7d)
	if !ok || len(js) < 10 {
		t.Fatalf("chart json ok=%v len=%d", ok, len(js))
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(js), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["repoA"] != "a/one" || payload["showMomentum"] != true {
		t.Errorf("payload = %v", payload)
	}
	_, okTotal := buildH2HChartJSON(db, "a/one", "b/two", h2h.IntervalTotal)
	if !okTotal {
		t.Error("expected total interval chart JSON")
	}
}
