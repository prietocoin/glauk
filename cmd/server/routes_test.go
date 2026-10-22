package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	comprobantesdomain "glauk/internal/comprobantes/domain"
	comprobanteshttp "glauk/internal/comprobantes/http"
	perfileshttp "glauk/internal/perfiles/http"
	perfilesservice "glauk/internal/perfiles/service"
	"glauk/internal/platform/config"
	"glauk/internal/reportes"
	reporteshttp "glauk/internal/reportes/http"
	tasashttp "glauk/internal/tasas/http"
	tasasservice "glauk/internal/tasas/service"
)

type stubComprobantes struct{ comprobanteshttp.Service }

func (stubComprobantes) List(context.Context,
	comprobantesdomain.ListFilters) ([]comprobantesdomain.Comprobante, error) {
	return []comprobantesdomain.Comprobante{}, nil
}

type stubSocios struct{}

func (stubSocios) SocioPorNombre(context.Context, string) (reportes.Socio, error) {
	return reportes.Socio{}, reportes.ErrSocioNoEncontrado
}

func (stubSocios) Socios(context.Context, string) ([]reportes.Socio, error) {
	return []reportes.Socio{}, nil
}

type stubMovimientos struct{}

func (stubMovimientos) Filtros(context.Context, string) (reportes.Filtros, error) {
	return reportes.Filtros{Roles: []string{"ADMIN"}, Entidades: []string{"GENERAL"}}, nil
}

func (stubMovimientos) Movimientos(context.Context, reportes.ConsultaMovimientos) ([]reportes.Movimiento, error) {
	return []reportes.Movimiento{}, nil
}

func (stubMovimientos) Resumen(context.Context, reportes.ConsultaMovimientos) (reportes.Resumen, error) {
	return reportes.Resumen{}, nil
}

type stubSender struct{}

func (stubSender) EnviarImagen(context.Context, reportes.MensajeImagen) (reportes.Recibo, error) {
	return reportes.Recibo{MessageID: "M", Estado: "SENT"}, nil
}

func (stubSender) EnviarTexto(context.Context, reportes.MensajeTexto) (reportes.Recibo, error) {
	return reportes.Recibo{MessageID: "M", Estado: "SENT"}, nil
}

func testDependencies(t *testing.T) *Dependencies {
	t.Helper()
	cfg := &config.Config{
		AppEnv: "test", Port: 3000,
		HTTP: config.HTTPConfig{BodyLimit: 20 << 20},
	}
	service := reportes.NewService(stubSocios{}, stubMovimientos{}, stubSender{}, reportes.Config{})
	module := &reportes.Module{
		Service: service, Idempotency: reportes.NewIdempotencyStore(time.Minute),
		RateLimiter: reportes.NewRateLimiter(600),
	}
	perfiles := perfileshttp.NewDeps(perfilesservice.New(nil, nil, nil))
	return &Dependencies{
		Config: cfg, Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Health: func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) },
		Ready:  func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) },
		Tasas: tasashttp.NewDeps(
			tasasservice.New(fakeTasasQuerier{}, nil, cfg), nil, time.Now),
		Perfiles:     perfiles,
		Comprobantes: comprobanteshttp.NewDeps(stubComprobantes{}, cfg),
		Reportes:     reporteshttp.NewHandler(module),
		Guard:        func(c *fiber.Ctx) error { return c.Next() },
	}
}

func testApp(t *testing.T) *fiber.App {
	t.Helper()
	deps := testDependencies(t)
	app := newApp(deps.Config, deps.Log)
	if err := registerRoutes(app, deps); err != nil {
		t.Fatalf("registerRoutes: %v", err)
	}
	return app
}

func request(t *testing.T, app *fiber.App, method, path string) (*nethttp.Response, []byte) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	body, _ := io.ReadAll(resp.Body)
	return resp, body
}

func TestNoDuplicateRoutes(t *testing.T) {
	app := testApp(t)
	seen := map[string]bool{}
	for _, route := range app.GetRoutes(true) {
		key := route.Method + " " + route.Path
		if seen[key] {
			t.Errorf("ruta duplicada: %s", key)
		}
		seen[key] = true
	}
}

func TestCriticalRoutesRegistered(t *testing.T) {
	app := testApp(t)
	seen := map[string]bool{}
	for _, route := range app.GetRoutes() {
		seen[route.Method+" "+route.Path] = true
	}
	for _, key := range []string{
		"GET /health", "GET /ready",
		"GET /api/v1/reportes/filtros", "GET /api/reportes/filtros",
		"GET /api/reportes", "GET /api/reportes/operaciones",
		"GET /api/tasas/ultimas", "GET /api/tasas/monitor",
		"POST /api/tasas/publicar", "POST /api/tasas/reenviar",
		"GET /api/preview-image/:identificador",
		"GET /api/v1/atenea/reportes/filtros", "GET /api/v1/atenea/reportes",
		"GET /api/v1/atenea/comprobantes", "GET /api/v1/atenea/tasas/ultimas",
	} {
		if !seen[key] {
			t.Errorf("falta la ruta %s", key)
		}
	}
}

func TestReportesResolutionAndDeprecation(t *testing.T) {
	app := testApp(t)
	resp, body := request(t, app, nethttp.MethodGet, "/api/reportes")
	if resp.StatusCode != 200 || !strings.HasPrefix(strings.TrimSpace(string(body)), "[") {
		t.Fatalf("/api/reportes debe ser el listado de Comprobantes: %d %s", resp.StatusCode, body)
	}

	resp, body = request(t, app, nethttp.MethodGet, "/api/reportes/filtros")
	var salida map[string]any
	if err := json.Unmarshal(body, &salida); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || salida["success"] != true || salida["roles"] == nil {
		t.Fatalf("/api/reportes/filtros debe ser Reportes: %d %v", resp.StatusCode, salida)
	}
	if resp.Header.Get("Deprecation") != "true" ||
		resp.Header.Get("Sunset") != sunsetAlias ||
		!strings.Contains(resp.Header.Get("Link"), "successor-version") {
		t.Fatalf("faltan cabeceras de deprecación: %v", resp.Header)
	}

	resp, body = request(t, app, nethttp.MethodGet, "/api/v1/reportes/filtros")
	if resp.StatusCode != 200 || !strings.Contains(string(body), `"data"`) {
		t.Fatalf("canónico: %d %s", resp.StatusCode, body)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	app := testApp(t)
	resp, body := request(t, app, nethttp.MethodGet, "/api/tasas/disparar")
	if resp.StatusCode != 405 || !strings.Contains(resp.Header.Get("Allow"), "POST") {
		t.Fatalf("405 esperado con Allow POST: %d %q %s", resp.StatusCode,
			resp.Header.Get("Allow"), body)
	}
}

func TestUnknownAPIStaysJSON(t *testing.T) {
	app := testApp(t)
	resp, body := request(t, app, nethttp.MethodGet, "/api/no-existe")
	if resp.StatusCode != 404 || !strings.Contains(resp.Header.Get("Content-Type"), "json") {
		t.Fatalf("API desconocida = %d %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	var envelope map[string]any
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("404 de API no es JSON: %v", err)
	}
	if envelope["success"] != false || envelope["code"] != "NOT_FOUND" {
		t.Fatalf("404 de API con envelope inesperado: %v", envelope)
	}
	if strings.Contains(string(body), "panel") {
		t.Fatalf("el 404 de API devolvió HTML: %s", body)
	}
}
