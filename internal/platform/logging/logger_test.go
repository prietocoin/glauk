package logging

import (
	"bytes"
	"strings"
	"testing"

	"glauk/internal/platform/config"
)

func testConfig(appEnv, level, format string) *config.Config {
	return &config.Config{
		AppEnv: appEnv,
		Log:    config.LogConfig{Level: level, Format: format},
	}
}

func TestProductionUsesJSONAndRedacts(t *testing.T) {
	var buffer bytes.Buffer
	logger := New(testConfig("production", "info", "auto"), &buffer)
	logger.Info("started", "key", MaskKey("supersecretkey"))

	output := buffer.String()
	if !strings.Contains(output, `"msg":"started"`) {
		t.Fatalf("salida no JSON: %s", output)
	}
	if strings.Contains(output, "supersecretkey") {
		t.Fatalf("se filtró la clave: %s", output)
	}
}

func TestDevelopmentUsesText(t *testing.T) {
	var buffer bytes.Buffer
	logger := New(testConfig("development", "debug", "auto"), &buffer)
	logger.Debug("trace", "jid", MaskJID("5491123456789@g.us"))

	output := buffer.String()
	if strings.HasPrefix(strings.TrimSpace(output), "{") {
		t.Fatalf("se esperaba texto: %s", output)
	}
	if !strings.Contains(output, "msg=trace") || !strings.Contains(output, "jid=54911****@g.us") {
		t.Fatalf("salida inesperada: %s", output)
	}
}

func TestLevelFilters(t *testing.T) {
	var buffer bytes.Buffer
	logger := New(testConfig("development", "warn", "text"), &buffer)
	logger.Info("hidden")
	logger.Warn("visible")

	output := buffer.String()
	if strings.Contains(output, "hidden") {
		t.Fatalf("el nivel info no debía emitirse: %s", output)
	}
	if !strings.Contains(output, "visible") {
		t.Fatalf("falta el nivel warn: %s", output)
	}
}
