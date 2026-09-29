package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/gghstats/internal/store"
)

func seedStuckFleet(t *testing.T, s *store.Store) string {
	t.Helper()
	now := time.Now().UTC()
	completed := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	// Stale relative to K=3: observed <= completed-3.
	observed := completed.AddDate(0, 0, -4).Format("2006-01-02")
	fetched := now
	for _, name := range []string{"o/r1", "o/r2", "o/r3"} {
		if err := s.UpsertRepo(name, "", 1, 0, 0, 0, 0, false, false, ""); err != nil {
			t.Fatal(err)
		}
		if err := s.RecordTrafficMetricSuccess(name, "clones", []store.DayRow{{Date: observed, Count: 10, Uniques: 2}}, fetched, observed, observed); err != nil {
			t.Fatal(err)
		}
		if err := s.RecordTrafficMetricSuccess(name, "views", []store.DayRow{{Date: observed, Count: 10, Uniques: 2}}, fetched, observed, observed); err != nil {
			t.Fatal(err)
		}
	}
	return observed
}

func TestHealthzUpstreamStaleActive(t *testing.T) {
	db := testStore(t)
	since := seedStuckFleet(t, db)
	h := New(Config{
		Store:               db,
		UpstreamStaleDays:   3,
		UpstreamStaleBanner: true,
	})
	req := httptest.NewRequest(http.MethodGet, HealthzPath, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var body struct {
		Status        string `json:"status"`
		UpstreamStale struct {
			Active        bool   `json:"active"`
			Since         string `json:"since"`
			DaysStuck     int    `json:"days_stuck"`
			StuckRepos    int    `json:"stuck_repos"`
			EligibleRepos int    `json:"eligible_repos"`
		} `json:"upstream_stale"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "ok" {
		t.Fatalf("status = %q", body.Status)
	}
	if !body.UpstreamStale.Active || body.UpstreamStale.Since != since || body.UpstreamStale.DaysStuck < 1 {
		t.Fatalf("upstream_stale = %+v want since=%s", body.UpstreamStale, since)
	}
}

func TestAPIReposUpstreamStaleWithBannerOff(t *testing.T) {
	db := testStore(t)
	seedStuckFleet(t, db)
	h := New(Config{
		Store:               db,
		APIToken:            "tok",
		UpstreamStaleDays:   3,
		UpstreamStaleBanner: false,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/repos", nil)
	req.Header.Set("x-api-token", "tok")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	us, ok := body["upstream_stale"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing upstream_stale: %v", body)
	}
	if us["active"] != true {
		t.Fatalf("active = %v", us["active"])
	}
}

func TestIndexUpstreamStaleBanner(t *testing.T) {
	db := testStore(t)
	since := seedStuckFleet(t, db)
	h := New(Config{
		Store:               db,
		UpstreamStaleDays:   3,
		UpstreamStaleBanner: true,
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	html := w.Body.String()
	if !strings.Contains(html, `id="upstream-stale-banner"`) {
		t.Fatal("missing upstream-stale banner")
	}
	if !strings.Contains(html, since) {
		t.Fatal("banner missing since date")
	}
	if !strings.Contains(html, "https://github.com/orgs/community/discussions?discussions_q=") {
		t.Fatal("banner missing Community discussions search help URL")
	}
	if !strings.Contains(html, "API") || !(strings.Contains(html, "Insights") && strings.Contains(html, "traffic")) {
		t.Fatal("banner help URL missing API Insights traffic query")
	}
	if strings.Contains(html, "208852") {
		t.Fatal("banner must not hard-code discussion #208852")
	}
}

func TestIndexUpstreamStaleBannerOff(t *testing.T) {
	db := testStore(t)
	seedStuckFleet(t, db)
	h := New(Config{
		Store:               db,
		UpstreamStaleDays:   3,
		UpstreamStaleBanner: false,
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if strings.Contains(w.Body.String(), `id="upstream-stale-banner"`) {
		t.Fatal("banner should be hidden when UpstreamStaleBanner=false")
	}
}

func TestRepoUpstreamStaleCue(t *testing.T) {
	db := testStore(t)
	since := seedStuckFleet(t, db)
	h := New(Config{
		Store:               db,
		UpstreamStaleDays:   3,
		UpstreamStaleBanner: true,
	})
	req := httptest.NewRequest(http.MethodGet, "/o/r1", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	html := w.Body.String()
	if !strings.Contains(html, `id="upstream-stale-cue"`) {
		t.Fatal("missing upstream-stale repo cue")
	}
	if !strings.Contains(html, since) {
		t.Fatal("repo cue missing since date")
	}
	if !strings.Contains(html, "https://github.com/orgs/community/discussions?discussions_q=") {
		t.Fatal("repo cue missing Community discussions search help URL")
	}
	if strings.Contains(html, "208852") {
		t.Fatal("repo cue must not hard-code discussion #208852")
	}
}

func TestRepoUpstreamStaleCueBannerOff(t *testing.T) {
	db := testStore(t)
	seedStuckFleet(t, db)
	h := New(Config{
		Store:               db,
		UpstreamStaleDays:   3,
		UpstreamStaleBanner: false,
	})
	req := httptest.NewRequest(http.MethodGet, "/o/r1", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if strings.Contains(w.Body.String(), `id="upstream-stale-cue"`) {
		t.Fatal("repo cue should be hidden when UpstreamStaleBanner=false")
	}
}
