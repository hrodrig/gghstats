package server

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// customCSSMaxBytes caps the served custom stylesheet size (self-hosted admin-controlled path).
const customCSSMaxBytes = 2 << 20 // 2 MiB

// ResolveCustomCSS validates an optional filesystem path from GGHSTATS_CUSTOM_CSS.
// It returns an absolute regular file path and a cache-busting query fragment (without leading "?").
// On any error or non-regular file, it returns "", "".
func ResolveCustomCSS(raw string) (absPath, cacheQuery string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	p := filepath.Clean(raw)
	if !filepath.IsAbs(p) {
		abs, err := filepath.Abs(p)
		if err != nil {
			return "", ""
		}
		p = abs
	}
	fi, err := os.Stat(p)
	if err != nil || !fi.Mode().IsRegular() {
		return "", ""
	}
	q := "v=" + strconv.FormatInt(fi.ModTime().Unix(), 10)
	return p, q
}

func handleCustomCSSEndpoint(absPath string) http.HandlerFunc {
	if absPath == "" {
		return func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}
	}
	return handleCustomCSS(absPath)
}

func handleCustomCSS(absPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(absPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()

		body, err := io.ReadAll(io.LimitReader(f, customCSSMaxBytes+1))
		if err != nil {
			slog.Error("read custom css", "path", absPath, "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if len(body) > customCSSMaxBytes {
			http.Error(w, "custom css file too large", http.StatusRequestEntityTooLarge)
			return
		}

		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		if _, err := w.Write(body); err != nil {
			slog.Error("write custom css", "error", err)
		}
	}
}
