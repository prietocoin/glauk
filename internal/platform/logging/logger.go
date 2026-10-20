package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"glauk/internal/platform/config"
)

// NewLogger builds the process-wide structured logger.
func NewLogger(cfg *config.Config) *slog.Logger {
	return New(cfg, os.Stdout)
}

// New lets tests inject a writer and keeps handler selection in one place.
// LOG_FORMAT=auto resolves to JSON in production and text elsewhere.
func New(cfg *config.Config, out io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(cfg.Log.Level)}
	format := strings.ToLower(strings.TrimSpace(cfg.Log.Format))
	if format == "auto" {
		if cfg.AppEnv == "production" {
			format = "json"
		} else {
			format = "text"
		}
	}
	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(out, opts)
	} else {
		handler = slog.NewTextHandler(out, opts)
	}
	return slog.New(handler)
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
