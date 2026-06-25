package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hrodrig/gghstats/internal/version"
)

// NewFormatLogHandler returns a slog.Handler that writes log lines in the format:
//
//	2026-06-24T01:02:43Z - gghstats - INFO - message key=value key=value
func NewFormatLogHandler(w io.Writer, level slog.Leveler) slog.Handler {
	if w == nil {
		w = os.Stderr
	}
	return &formatHandler{
		w:     w,
		level: level,
		mu:    sync.Mutex{},
	}
}

type formatHandler struct {
	w     io.Writer
	level slog.Leveler
	mu    sync.Mutex
}

func (h *formatHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *formatHandler) Handle(_ context.Context, r slog.Record) error {
	ts := r.Time.UTC().Format(time.RFC3339)
	level := strings.ToUpper(r.Level.String())
	msg := r.Message

	// Collect attributes as key=value pairs
	attrs := make([]string, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		if a.Equal(slog.Attr{}) {
			return true
		}
		attrs = append(attrs, fmt.Sprintf("%s=%v", a.Key, a.Value.Any()))
		return true
	})

	var line string
	if len(attrs) > 0 {
		line = fmt.Sprintf("%s - %s - %s - %s %s\n", ts, version.AppName, level, msg, strings.Join(attrs, " "))
	} else {
		line = fmt.Sprintf("%s - %s - %s - %s\n", ts, version.AppName, level, msg)
	}

	h.mu.Lock()
	_, err := h.w.Write([]byte(line))
	h.mu.Unlock()
	return err
}

func (h *formatHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h // not used in this project
}

func (h *formatHandler) WithGroup(name string) slog.Handler {
	return h // not used in this project
}

var _ slog.Handler = (*formatHandler)(nil)
