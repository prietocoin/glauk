package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func healthApp(deps HealthDeps) *fiber.App {
	app := fiber.New()
	app.Get("/health", Health(deps))
	app.Get("/ready", Ready(deps))
	return app
}

func healthBody(t *testing.T, app *fiber.App, target string) (int, map[string]any) {
	t.Helper()
	response, err := app.Test(httptest.NewRequest(http.MethodGet, target, nil))
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(responseBody(t, response)), &body); err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, body
}

func TestHealthAndReadyContracts(t *testing.T) {
	deps := HealthDeps{
		Database: func(context.Context) error { return nil },
		Evolution: func() (string, bool) {
			return "Glauk", true
		},
		Version: "1.0.0",
		Started: time.Now().Add(-2 * time.Second),
	}
	app := healthApp(deps)

	status, body := healthBody(t, app, "/health")
	if status != 200 || body["status"] != "ok" || body["service"] != "glauk" ||
		body["version"] != "1.0.0" {
		t.Fatalf("health inesperado: %d %#v", status, body)
	}
	checks := body["checks"].(map[string]any)
	database := checks["database"].(map[string]any)
	if database["status"] != "ok" {
		t.Fatalf("database = %#v", database)
	}
	evolution := checks["evolution"].(map[string]any)
	if evolution["status"] != "simulated" || evolution["instance"] != "Glauk" {
		t.Fatalf("evolution = %#v", evolution)
	}

	status, body = healthBody(t, app, "/ready")
	if status != 200 {
		t.Fatalf("ready status = %d", status)
	}
	if _, exists := body["version"]; exists {
		t.Fatal("ready reveló la versión")
	}
	evolution = body["checks"].(map[string]any)["evolution"].(map[string]any)
	if _, exists := evolution["instance"]; exists {
		t.Fatal("ready reveló la instancia de Evolution")
	}
}

func TestHealthDegradedDoesNotLeakCause(t *testing.T) {
	app := healthApp(HealthDeps{
		Database: func(context.Context) error {
			return errors.New("postgres://user:secret@internal/glauk")
		},
		Started: time.Now(),
	})
	for _, target := range []string{"/health", "/ready"} {
		status, body := healthBody(t, app, target)
		if status != 503 || body["status"] != "degraded" {
			t.Fatalf("%s inesperado: %d %#v", target, status, body)
		}
		raw, _ := json.Marshal(body)
		for _, fragment := range []string{"postgres://", "secret", "internal"} {
			if strings.Contains(string(raw), fragment) {
				t.Fatalf("%s filtró detalles: %s", target, raw)
			}
		}
	}
}
