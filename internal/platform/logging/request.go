package logging

import (
	"context"
	"log/slog"
	"time"
)

// Request contains only operational metadata. Bodies, query strings with
// tokens, headers, and credentials are deliberately absent.
type Request struct {
	RequestID string
	Method    string
	Path      string
	Status    int
	Duration  time.Duration
	Bytes     int64
	Tenant    string
}

// Attrs returns the canonical request fields required by §2.8.
func (r Request) Attrs() []slog.Attr {
	return []slog.Attr{
		slog.String("request_id", r.RequestID),
		slog.String("method", r.Method),
		slog.String("path", r.Path),
		slog.Int("status", r.Status),
		slog.Int64("duration_ms", r.Duration.Milliseconds()),
		slog.Int64("bytes", r.Bytes),
		slog.String("tenant", r.Tenant),
	}
}

// Log emits one structured request line at a level derived from the status.
func (r Request) Log(logger *slog.Logger) {
	logger.LogAttrs(context.Background(), levelForStatus(r.Status), "http_request", r.Attrs()...)
}

func levelForStatus(status int) slog.Level {
	switch {
	case status >= 500:
		return slog.LevelError
	case status >= 400:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}
