package server

import (
	"net/http/httptest"
	"testing"

	"github.com/hrodrig/gghstats/internal/i18n"
)

func TestRequestScheme(t *testing.T) {
	t.Run("forwarded proto https", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		if got := requestScheme(req); got != "https" {
			t.Fatalf("requestScheme = %q, want https", got)
		}
		if !requestIsHTTPS(req) {
			t.Fatal("requestIsHTTPS = false, want true")
		}
	})

	t.Run("plain http", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		if got := requestScheme(req); got != "http" {
			t.Fatalf("requestScheme = %q, want http", got)
		}
		if requestIsHTTPS(req) {
			t.Fatal("requestIsHTTPS = true, want false")
		}
	})
}

func TestLocaleCookieSecure(t *testing.T) {
	t.Run("secure behind proxy", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?lang=es", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		w := httptest.NewRecorder()
		maybeSetLocaleCookie(w, req, Config{DefaultLocale: "en", EnabledLocales: []string{"en", "es"}})
		resp := w.Result()
		defer resp.Body.Close()
		var secure bool
		for _, c := range resp.Cookies() {
			if c.Name == i18n.CookieName {
				secure = c.Secure
				break
			}
		}
		if !secure {
			t.Fatal("locale cookie Secure = false, want true with X-Forwarded-Proto https")
		}
	})

	t.Run("always secure", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?lang=es", nil)
		w := httptest.NewRecorder()
		maybeSetLocaleCookie(w, req, Config{DefaultLocale: "en", EnabledLocales: []string{"en", "es"}})
		resp := w.Result()
		defer resp.Body.Close()
		for _, c := range resp.Cookies() {
			if c.Name == i18n.CookieName && !c.Secure {
				t.Fatal("locale cookie Secure = false, want true")
			}
		}
	})
}
