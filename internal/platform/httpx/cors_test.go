package httpx

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/platform/config"
)

const (
	glaukUIOrigin       = "https://automat-glauk-ui.fyi6ur.easypanel.host"
	glaukJairokovOrigin = "https://glauk.jairokov.com"
)

func corsTestApp(t *testing.T) *fiber.App {
	t.Helper()
	cfg := &config.Config{
		AppEnv: "test",
		Port:   3000,
		HTTP: config.HTTPConfig{
			BodyLimit:      1 << 20,
			ReadTimeout:    time.Second,
			WriteTimeout:   time.Second,
			IdleTimeout:    time.Second,
			AllowedOrigins: []string{glaukJairokovOrigin, glaukUIOrigin, "http://localhost:*"},
		},
	}
	app := NewApp(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	app.Get("/api/ping", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	return app
}

func responseBody(t *testing.T, response *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	return string(body)
}

func corsRequest(t *testing.T, app *fiber.App, method, origin string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, "/api/ping", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if method == http.MethodOptions {
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		req.Header.Set("Access-Control-Request-Headers", "authorization,content-type,x-requested-with")
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, origin, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func TestCORSPreflightAllowsGlaukUI(t *testing.T) {
	resp := corsRequest(t, corsTestApp(t), http.MethodOptions, glaukUIOrigin)
	if resp.StatusCode != fiber.StatusNoContent && resp.StatusCode != fiber.StatusOK {
		t.Fatalf("preflight status = %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != glaukUIOrigin {
		t.Fatalf("allow-origin = %q", got)
	}
	methods := resp.Header.Get("Access-Control-Allow-Methods")
	for _, method := range []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"} {
		if !strings.Contains(methods, method) {
			t.Fatalf("allow-methods no incluye %s: %q", method, methods)
		}
	}
	headers := resp.Header.Get("Access-Control-Allow-Headers")
	for _, header := range []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Tenant-ID", "X-Requested-With"} {
		if !strings.Contains(headers, header) {
			t.Fatalf("allow-headers no incluye %s: %q", header, headers)
		}
	}
}

func TestCORSPreflightAllowsGlaukJairokov(t *testing.T) {
	resp := corsRequest(t, corsTestApp(t), http.MethodOptions, glaukJairokovOrigin)
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != glaukJairokovOrigin {
		t.Fatalf("allow-origin = %q", got)
	}
}

func TestCORSAllowsLocalhostAnyPort(t *testing.T) {
	for _, origin := range []string{"http://localhost", "http://localhost:5173", "http://localhost:8080"} {
		resp := corsRequest(t, corsTestApp(t), http.MethodGet, origin)
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("%s status = %d", origin, resp.StatusCode)
		}
		if got := resp.Header.Get("Access-Control-Allow-Origin"); got != origin {
			t.Fatalf("%s allow-origin = %q", origin, got)
		}
	}
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	resp := corsRequest(t, corsTestApp(t), http.MethodOptions, "https://evil.example")
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("origen desconocido fue aceptado: %q", resp.Header.Get("Access-Control-Allow-Origin"))
	}
}
