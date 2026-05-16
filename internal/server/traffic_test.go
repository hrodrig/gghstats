package server

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAPIRepoTrafficUnauthorized(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db, APIToken: "secret"})

	req := httptest.NewRequest("GET", "/api/v1/repos/a/b/traffic", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestAPIRepoTrafficDisabledWithoutToken(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db, APIToken: ""})

	req := httptest.NewRequest("GET", "/api/v1/repos/a/b/traffic", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestAPIRepoTrafficReturnsSeries(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	today := time.Now().UTC().Format("2006-01-02")
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	db.UpsertView("a/b", today, 50, 20)
	db.UpsertClone("a/b", yesterday, 5, 2)
	db.UpsertClone("a/b", today, 12, 4)

	handler := New(Config{Store: db, APIToken: "secret"})
	req := httptest.NewRequest("GET", "/api/v1/repos/a/b/traffic?days=30", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d body=%q", w.Code, w.Body.String())
	}
	var resp repoTrafficResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Name != "a/b" || resp.Days != 30 {
		t.Fatalf("resp = %+v", resp)
	}
	if len(resp.Clones) != 2 || len(resp.Views) != 1 {
		t.Fatalf("clones=%d views=%d", len(resp.Clones), len(resp.Views))
	}
	if resp.Clones[0].Date != yesterday || resp.Clones[0].Count != 5 {
		t.Fatalf("clones[0] = %+v, want date %s count 5", resp.Clones[0], yesterday)
	}
}

func TestAPIRepoTrafficNotFound(t *testing.T) {
	db := testStore(t)
	handler := New(Config{Store: db, APIToken: "secret"})

	req := httptest.NewRequest("GET", "/api/v1/repos/missing/r/traffic", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestAPIRepoTrafficInvalidDays(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	handler := New(Config{Store: db, APIToken: "secret"})

	req := httptest.NewRequest("GET", "/api/v1/repos/a/b/traffic?days=abc", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid days") {
		t.Fatalf("body = %q", w.Body.String())
	}
}

func TestAPIRepoTrafficDaysExceedsMax(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	handler := New(Config{Store: db, APIToken: "secret"})

	req := httptest.NewRequest("GET", "/api/v1/repos/a/b/traffic?days=99999", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestAPIRepoTrafficEmptyHistory(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	handler := New(Config{Store: db, APIToken: "secret"})

	req := httptest.NewRequest("GET", "/api/v1/repos/a/b/traffic?days=7", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var resp repoTrafficResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Clones) != 0 || len(resp.Views) != 0 {
		t.Fatalf("expected empty series, got clones=%d views=%d", len(resp.Clones), len(resp.Views))
	}
}

func TestParseTrafficDays(t *testing.T) {
	got, err := parseTrafficDays("")
	if err != nil || got != defaultTrafficDays {
		t.Fatalf("default: got %d err %v", got, err)
	}
	if _, err := parseTrafficDays("x"); err == nil {
		t.Fatal("expected error for non-numeric")
	}
	if _, err := parseTrafficDays("-1"); err == nil {
		t.Fatal("expected error for negative")
	}
}

func TestTrafficDateRangeUTC(t *testing.T) {
	from, to, err := trafficDateRangeUTC(7, "2026-01-01", true)
	if err != nil {
		t.Fatal(err)
	}
	if to == "" || from == "" {
		t.Fatalf("from=%q to=%q", from, to)
	}
	from0, to0, err := trafficDateRangeUTC(0, "", false)
	if err != nil || from0 != to0 {
		t.Fatalf("no extent: from=%q to=%q err=%v", from0, to0, err)
	}
}

func TestAPIRepoTrafficAllTime(t *testing.T) {
	db := testStore(t)
	db.UpsertRepo("a/b", "", 0, 0, 0, 0, 0, false, false, "")
	db.UpsertClone("a/b", "2026-01-01", 1, 1)
	db.UpsertView("a/b", "2026-02-01", 2, 1)

	handler := New(Config{Store: db, APIToken: "secret"})
	req := httptest.NewRequest("GET", "/api/v1/repos/a/b/traffic?days=0", nil)
	req.Header.Set("x-api-token", "secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var resp repoTrafficResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.From != "2026-01-01" {
		t.Errorf("from = %q, want 2026-01-01", resp.From)
	}
	if len(resp.Clones) != 1 || len(resp.Views) != 1 {
		t.Fatalf("clones=%d views=%d", len(resp.Clones), len(resp.Views))
	}
}
