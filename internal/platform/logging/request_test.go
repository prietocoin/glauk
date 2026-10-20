package logging

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"glauk/internal/platform/config"
)

func TestRequestLogsOnlyOperationalFields(t *testing.T) {
	var buffer bytes.Buffer
	logger := New(&config.Config{
		AppEnv: "development",
		Log:    config.LogConfig{Level: "info", Format: "text"},
	}, &buffer)

	Request{
		RequestID: "req-1",
		Method:    "GET",
		Path:      "/health",
		Status:    200,
		Duration:  12 * time.Millisecond,
		Bytes:     2,
		Tenant:    "tasas",
	}.Log(logger)

	output := buffer.String()
	for _, field := range []string{
		"request_id=req-1", "method=GET", "path=/health", "status=200",
		"duration_ms=12", "bytes=2", "tenant=tasas",
	} {
		if !strings.Contains(output, field) {
			t.Errorf("falta %s en %s", field, output)
		}
	}
}

func TestRequestLevelUsesStatus(t *testing.T) {
	if levelForStatus(500) != 8 {
		t.Fatalf("500 debe ser ERROR, got %v", levelForStatus(500))
	}
	if levelForStatus(404) != 4 {
		t.Fatalf("404 debe ser WARN, got %v", levelForStatus(404))
	}
	if levelForStatus(204) != 0 {
		t.Fatalf("204 debe ser INFO, got %v", levelForStatus(204))
	}
}
