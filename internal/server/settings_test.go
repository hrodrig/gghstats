package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsPageRequiresAPITokenAndIsRedacted(t *testing.T) {
	db := testStore(t)
	h := New(Config{
		Store:          db,
		APIToken:       "api-secret-must-not-render",
		DisableMetrics: true,
		Settings: SettingsSnapshot{
			RuntimeMode:           "Production",
			Host:                  "127.0.0.1",
			Port:                  "8080",
			DatabaseReady:         true,
			GithubTokenConfigured: true,
			APITokenConfigured:    true,
			FilterAll:             false,
			IncludePrivate:        false,
			ReportPrivate:         false,
			SyncInterval:          "1h0m0s",
			SyncOnStartup:         true,
			SyncWorkers:           4,
			BadgePublic:           true,
			MetricsEnabled:        true,
			RateLimitEnabled:      true,
			CompactNumbers:        false,
			DefaultLocale:         "en",
			EnabledLocales:        []string{"en", "es"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if strings.Contains(w.Body.String(), "api-secret-must-not-render") || !strings.Contains(w.Body.String(), "Continue to settings") {
		t.Fatal("unauthenticated settings gate should be safe and actionable")
	}

	req = httptest.NewRequest(http.MethodGet, "/settings", nil)
	req.Header.Set("x-api-token", "api-secret-must-not-render")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	for _, forbidden := range []string{"api-secret-must-not-render", "ghp_example_secret", `C:\secret\gghstats.db`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("settings page leaked %q", forbidden)
		}
	}
	for _, expected := range []string{"Settings and safety", "Read only", "Custom filter configured", "GitHub access"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("settings page missing %q", expected)
		}
	}
}

func TestSettingsBrowserSessionAllowsPageAndSafeUpdate(t *testing.T) {
	db := testStore(t)
	manager, err := NewSettingsManager("", EditableSettings{
		DefaultLocale: "en",
	}, []string{"en", "de"})
	if err != nil {
		t.Fatal(err)
	}
	h := New(Config{
		Store:           db,
		APIToken:        "admin-secret",
		SettingsManager: manager,
		DefaultLocale:   "en",
		EnabledLocales:  []string{"en", "de"},
		DisableMetrics:  true,
	})

	sessionReq := httptest.NewRequest(http.MethodPost, "/api/v1/settings/session", nil)
	sessionReq.Header.Set("x-api-token", "admin-secret")
	sessionWriter := httptest.NewRecorder()
	h.ServeHTTP(sessionWriter, sessionReq)
	if sessionWriter.Code != http.StatusOK {
		t.Fatalf("session status = %d, want %d", sessionWriter.Code, http.StatusOK)
	}
	cookies := sessionWriter.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != settingsSessionCookieName {
		t.Fatalf("session cookies = %+v, want one settings session cookie", cookies)
	}
	if cookies[0].Value == "" || strings.Contains(cookies[0].Value, "admin-secret") || !cookies[0].HttpOnly {
		t.Fatalf("session cookie should be opaque and HttpOnly: %+v", cookies[0])
	}

	get := httptest.NewRequest(http.MethodGet, "/settings", nil)
	get.AddCookie(cookies[0])
	page := httptest.NewRecorder()
	h.ServeHTTP(page, get)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Settings and safety") {
		t.Fatalf("session-authenticated page status/body = %d/%q", page.Code, page.Body.String()[:min(120, len(page.Body.String()))])
	}

	post := httptest.NewRequest(http.MethodPost, "/api/v1/settings", bytes.NewBufferString(`{"compact_numbers":true}`))
	post.Header.Set("Content-Type", "application/json")
	post.AddCookie(cookies[0])
	updated := httptest.NewRecorder()
	h.ServeHTTP(updated, post)
	if updated.Code != http.StatusOK || !manager.Snapshot().CompactNumbers {
		t.Fatalf("session-authenticated update status/value = %d/%+v", updated.Code, manager.Snapshot())
	}
}

func TestSettingsUpdateRequiresAPITokenAndPersistsAllowlistedValues(t *testing.T) {
	db := testStore(t)
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	manager, err := NewSettingsManager(settingsPath, EditableSettings{
		DefaultLocale:  "en",
		CompactNumbers: false,
	}, []string{"en", "de"})
	if err != nil {
		t.Fatal(err)
	}
	h := New(Config{
		Store:           db,
		APIToken:        "admin-secret",
		SettingsManager: manager,
		DefaultLocale:   "en",
		EnabledLocales:  []string{"en", "de"},
		DisableMetrics:  true,
	})

	post := func(token, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("x-api-token", token)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}

	if got := post("", `{"compact_numbers":true}`).Code; got != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d, want %d", got, http.StatusUnauthorized)
	}
	if got := post("admin-secret", `{"compact_numbers":true,"default_locale":"de"}`).Code; got != http.StatusOK {
		t.Fatalf("valid update status = %d, want %d", got, http.StatusOK)
	}
	values := manager.Snapshot()
	if !values.CompactNumbers || values.DefaultLocale != "de" {
		t.Fatalf("updated values = %+v", values)
	}
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "admin-secret") || strings.Contains(string(raw), settingsPath) {
		t.Fatalf("settings file contains protected data: %s", raw)
	}
	if got := post("admin-secret", `{"database":"/secret/path"}`).Code; got != http.StatusBadRequest {
		t.Fatalf("unknown field status = %d, want %d", got, http.StatusBadRequest)
	}
	if got := post("admin-secret", `{"default_locale":"bg"}`).Code; got != http.StatusBadRequest {
		t.Fatalf("disabled locale status = %d, want %d", got, http.StatusBadRequest)
	}
}

func TestSettingsUpdateAllowsSafePreferencesOnLoopbackWithoutToken(t *testing.T) {
	db := testStore(t)
	manager, err := NewSettingsManager(filepath.Join(t.TempDir(), "settings.json"), EditableSettings{
		DefaultLocale: "en",
	}, []string{"en", "de"})
	if err != nil {
		t.Fatal(err)
	}
	h := New(Config{
		Store:             db,
		SettingsManager:   manager,
		LocalOnlySettings: true,
		DefaultLocale:     "en",
		EnabledLocales:    []string{"en", "de"},
		DisableMetrics:    true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings", bytes.NewBufferString(`{"compact_numbers":true,"default_locale":"de"}`))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("local update status = %d, want %d", w.Code, http.StatusOK)
	}
	values := manager.Snapshot()
	if !values.CompactNumbers || values.DefaultLocale != "de" {
		t.Fatalf("updated values = %+v", values)
	}
	get := httptest.NewRequest(http.MethodGet, "/settings?lang=en", nil)
	get.RemoteAddr = "[::1]:12345"
	page := httptest.NewRecorder()
	h.ServeHTTP(page, get)
	if page.Code != http.StatusOK {
		t.Fatalf("local settings page status = %d, want %d", page.Code, http.StatusOK)
	}
	body := page.Body.String()
	if !strings.Contains(body, "Local-only controls") || !strings.Contains(body, `data-local-only="true"`) {
		t.Fatal("local settings page should expose token-free safe controls")
	}
}

func TestSettingsUpdateIsUnavailableWithoutTokenOnNonLoopback(t *testing.T) {
	db := testStore(t)
	manager, err := NewSettingsManager("", EditableSettings{DefaultLocale: "en"}, []string{"en"})
	if err != nil {
		t.Fatal(err)
	}
	h := New(Config{Store: db, SettingsManager: manager, DisableMetrics: true})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings", bytes.NewBufferString(`{"compact_numbers":true}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unprotected remote update status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestSettingsUpdateDeniesLoopbackWhenForwardedHeadersPresent(t *testing.T) {
	db := testStore(t)
	manager, err := NewSettingsManager(filepath.Join(t.TempDir(), "settings.json"), EditableSettings{
		DefaultLocale: "en",
	}, []string{"en"})
	if err != nil {
		t.Fatal(err)
	}
	h := New(Config{
		Store:             db,
		SettingsManager:   manager,
		LocalOnlySettings: true,
		DisableMetrics:    true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings", bytes.NewBufferString(`{"compact_numbers":true}`))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("loopback+XFF status = %d, want %d (local-only denied)", w.Code, http.StatusNotFound)
	}
}

func TestCompactNumbersUseLiveSettingsSnapshot(t *testing.T) {
	db := testStore(t)
	if err := db.UpsertRepo("a/b", "", 1500, 0, 0, 0, 0, false, false, ""); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertClone("a/b", "2026-03-18", 1500, 1200); err != nil {
		t.Fatal(err)
	}
	manager, err := NewSettingsManager("", EditableSettings{
		DefaultLocale:  "en",
		CompactNumbers: false,
	}, []string{"en"})
	if err != nil {
		t.Fatal(err)
	}
	h := New(Config{
		Store:           db,
		SettingsManager: manager,
		DisableMetrics:  true,
	})

	render := func() string {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("index status = %d, want %d", w.Code, http.StatusOK)
		}
		return w.Body.String()
	}

	body := render()
	if !strings.Contains(body, `window.gghstatsNumberFormat = {"compact":false,"locale":"en"}`) {
		t.Fatal("initial page should use non-compact number formatting")
	}
	if !strings.Contains(body, ">1,500<") {
		t.Fatal("initial clone chart statistics should use grouped numbers")
	}

	compact := true
	if _, err := manager.Update(SettingsUpdate{CompactNumbers: &compact}); err != nil {
		t.Fatal(err)
	}
	body = render()
	if !strings.Contains(body, `window.gghstatsNumberFormat = {"compact":true,"locale":"en"}`) {
		t.Fatal("page should use the updated compact number setting without restart")
	}
	if !strings.Contains(body, ">1.5k<") {
		t.Fatal("updated clone chart statistics should use compact numbers")
	}
}

func TestSettingsRejectsNonLoopbackRequestsWhenLocalOnly(t *testing.T) {
	db := testStore(t)
	manager, err := NewSettingsManager("", EditableSettings{DefaultLocale: "en"}, []string{"en"})
	if err != nil {
		t.Fatal(err)
	}
	h := New(Config{Store: db, SettingsManager: manager, LocalOnlySettings: true, DisableMetrics: true})

	get := httptest.NewRequest(http.MethodGet, "/settings", nil)
	get.RemoteAddr = "192.0.2.10:12345"
	getWriter := httptest.NewRecorder()
	h.ServeHTTP(getWriter, get)
	if getWriter.Code != http.StatusNotFound {
		t.Fatalf("non-loopback GET status = %d, want %d", getWriter.Code, http.StatusNotFound)
	}

	post := httptest.NewRequest(http.MethodPost, "/api/v1/settings", bytes.NewBufferString(`{"compact_numbers":true}`))
	post.RemoteAddr = "192.0.2.10:12345"
	postWriter := httptest.NewRecorder()
	h.ServeHTTP(postWriter, post)
	if postWriter.Code != http.StatusNotFound {
		t.Fatalf("non-loopback POST status = %d, want %d", postWriter.Code, http.StatusNotFound)
	}
}
