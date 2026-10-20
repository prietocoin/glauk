package httpx

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/platform/config"
)

func staticConfig(t *testing.T) *config.Config {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"index.html":               "<html>LANZADOR</html>",
		"tasas/index.html":         "<html>TASAS</html>",
		"perfiles/index.html":      "<html>PERFILES</html>",
		"comprobantes/index.html":  "<html>COMPROBANTES</html>",
		"reportes/index.html":      "<html>REPORTES</html>",
		"shared/app.js":            "shared",
		"shared/brand/favicon.svg": "<svg/>",
		"tasas/main.js":            "main",
		"tasas/main.a1b2c3d4.js":   "hashed",
		"tasas/sw.js":              "sw",
		"tasas/manifest.json":      "{}",
	}
	for name, body := range files {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return &config.Config{
		PublicDir: root,
		HTTP: config.HTTPConfig{
			BodyLimit:       1 << 20,
			ReadTimeout:     time.Second,
			WriteTimeout:    time.Second,
			IdleTimeout:     time.Second,
			AllowedOrigins:  []string{"*"},
			HostRoutingMode: "both",
			DefaultTenant:   "comprobantes",
		},
		Tenants: config.TenantConfig{
			Tasas:        []string{"tasas.localhost"},
			Perfiles:     []string{"perfiles.localhost"},
			Comprobantes: []string{"comprobantes.localhost"},
			Reportes:     []string{"reportes.localhost"},
		},
	}
}

func staticApp(t *testing.T, mutate func(*config.Config)) *fiber.App {
	t.Helper()
	cfg := staticConfig(t)
	if mutate != nil {
		mutate(cfg)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := NewApp(cfg, logger)
	RegisterStatic(app, cfg)
	return app
}

func request(t *testing.T, app *fiber.App, target, host string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if host != "" {
		req.Host = host
	}
	response, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", host, target, err)
	}
	return response
}

func responseBody(t *testing.T, response *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	return string(body)
}

func TestStaticHostAndPathRouting(t *testing.T) {
	app := staticApp(t, nil)
	for _, tc := range []struct {
		name, target, host, want string
	}{
		{"host", "/", "tasas.localhost", "TASAS"},
		{"path", "/tasas/", "localhost", "TASAS"},
		{"path perfiles", "/perfiles/", "localhost", "PERFILES"},
		{"host gana al path", "/perfiles/", "tasas.localhost", "TASAS"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := request(t, app, tc.target, tc.host)
			if response.StatusCode != 200 {
				t.Fatalf("status = %d", response.StatusCode)
			}
			if body := responseBody(t, response); !strings.Contains(body, tc.want) {
				t.Fatalf("body = %q", body)
			}
		})
	}
}

func TestAPIAndTraversalNeverReachSPA(t *testing.T) {
	app := staticApp(t, nil)
	response := request(t, app, "/api/inexistente", "localhost")
	if response.StatusCode != 404 {
		t.Fatalf("api status = %d", response.StatusCode)
	}
	if !strings.Contains(response.Header.Get("Content-Type"), "json") {
		t.Fatal("la API 404 no es JSON")
	}
	body := responseBody(t, response)
	if !strings.Contains(body, `"code":"NOT_FOUND"`) || strings.Contains(body, "<html>") {
		t.Fatalf("body = %s", body)
	}

	for _, target := range []string{"/../../etc/passwd", "/assets/..%2f..%2fetc/passwd"} {
		response = request(t, app, target, "localhost")
		if response.StatusCode != 403 {
			t.Fatalf("%s status = %d", target, response.StatusCode)
		}
		if body := responseBody(t, response); !strings.Contains(body, "FORBIDDEN") {
			t.Fatalf("%s body = %s", target, body)
		}
	}
}

func TestMissingAssetAndSPAFallback(t *testing.T) {
	app := staticApp(t, nil)
	response := request(t, app, "/tasas/missing.js", "localhost")
	if response.StatusCode != 404 {
		t.Fatalf("asset status = %d", response.StatusCode)
	}
	if body := responseBody(t, response); strings.Contains(body, "TASAS") {
		t.Fatal("un asset inexistente cayó al SPA")
	}

	response = request(t, app, "/tasas/ruta-sin-extension", "localhost")
	if response.StatusCode != 200 {
		t.Fatalf("spa status = %d", response.StatusCode)
	}
	if body := responseBody(t, response); !strings.Contains(body, "TASAS") {
		t.Fatalf("body = %q", body)
	}
}

func TestStaticCachePolicy(t *testing.T) {
	app := staticApp(t, nil)
	for _, tc := range []struct{ target, cache string }{
		{"/tasas/main.a1b2c3d4.js", "public, max-age=31536000, immutable"},
		{"/tasas/main.js", "no-cache"},
		{"/tasas/", "no-cache"},
		{"/shared/app.js", "public, max-age=3600"},
	} {
		response := request(t, app, tc.target, "localhost")
		if response.StatusCode != 200 {
			t.Fatalf("%s status = %d", tc.target, response.StatusCode)
		}
		if got := response.Header.Get("Cache-Control"); got != tc.cache {
			t.Fatalf("%s cache = %q", tc.target, got)
		}
		_ = responseBody(t, response)
	}
}

func TestUnknownHostAndDefaultNone(t *testing.T) {
	hostApp := staticApp(t, func(cfg *config.Config) {
		cfg.HTTP.HostRoutingMode = "host"
	})
	if response := request(t, hostApp, "/", "desconocido.localhost"); response.StatusCode != 404 {
		t.Fatalf("host desconocido status = %d", response.StatusCode)
	}

	noneApp := staticApp(t, func(cfg *config.Config) {
		cfg.HTTP.HostRoutingMode = "path"
		cfg.HTTP.DefaultTenant = "none"
	})
	response := request(t, noneApp, "/", "")
	if response.StatusCode != 200 {
		t.Fatalf("lanzador status = %d", response.StatusCode)
	}
	if body := responseBody(t, response); !strings.Contains(body, "LANZADOR") {
		t.Fatalf("body = %q", body)
	}
	if response := request(t, noneApp, "/otra", ""); response.StatusCode != 404 {
		t.Fatalf("default none status = %d", response.StatusCode)
	}
}
