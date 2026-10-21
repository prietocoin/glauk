package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/reportes"
)

func nuevoAppModulo(t *testing.T, module *reportes.Module) *fiber.App {
	t.Helper()
	app := fiber.New()
	Register(app, NewHandler(module), nil)
	return app
}

func pedirConHeader(t *testing.T, app *fiber.App, method, path, body, key string) (*nethttp.Response, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test %s %s: %v", method, path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	data, _ := io.ReadAll(resp.Body)
	var out map[string]any
	if len(data) > 0 {
		_ = json.Unmarshal(data, &out)
	}
	return resp, out
}

func TestEnvioConIdempotenciaYDuplicado(t *testing.T) {
	service := reportes.NewService(fakeSocios{}, fakeMovimientos{}, fakeSender{}, reportes.Config{})
	module := &reportes.Module{
		Service:     service,
		Idempotency: reportes.NewIdempotencyStore(time.Minute),
		RateLimiter: reportes.NewRateLimiter(60),
	}
	app := nuevoAppModulo(t, module)
	body := `{"socio":"FUNDABLOCK","base64":"` + jpegBase64(t) + `"}`

	resp, out := pedirConHeader(t, app, nethttp.MethodPost,
		"/api/v1/reportes/imagen", body, "clave-1234")
	if resp.StatusCode != 200 {
		t.Fatalf("primer envío: %d %v", resp.StatusCode, out)
	}
	data := out["data"].(map[string]any)
	if data["duplicate"] != false {
		t.Fatalf("primer envío duplicado: %v", data)
	}
	resp, out = pedirConHeader(t, app, nethttp.MethodPost,
		"/api/v1/reportes/imagen", body, "clave-1234")
	data = out["data"].(map[string]any)
	if resp.StatusCode != 200 || data["duplicate"] != true {
		t.Fatalf("segundo envío: %d %v", resp.StatusCode, out)
	}
}

func TestEnvioEnCursoYLimiteIdempotencia(t *testing.T) {
	store := reportes.NewIdempotencyStore(time.Minute)
	service := reportes.NewService(fakeSocios{}, fakeMovimientos{}, fakeSender{}, reportes.Config{})
	app := nuevoAppModulo(t, &reportes.Module{Service: service, Idempotency: store})
	if _, created, _ := store.Begin("clave-1234"); !created {
		t.Fatal("no reservó la clave")
	}
	body := `{"socio":"FUNDABLOCK","base64":"` + jpegBase64(t) + `"}`
	resp, out := pedirConHeader(t, app, nethttp.MethodPost,
		"/api/v1/reportes/imagen", body, "clave-1234")
	if resp.StatusCode != 409 || out["code"] != "ENVIO_EN_CURSO" {
		t.Fatalf("en curso: %d %v", resp.StatusCode, out)
	}
}

func TestEnvioSinIdempotenciaYRateLimit(t *testing.T) {
	service := reportes.NewService(fakeSocios{}, fakeMovimientos{}, fakeSender{}, reportes.Config{})
	module := &reportes.Module{Service: service, RateLimiter: reportes.NewRateLimiter(60)}
	app := nuevoAppModulo(t, module)
	body := `{"socio":"FUNDABLOCK","texto":"hola"}`
	for i := 0; i < 10; i++ {
		resp, out := pedirConHeader(t, app, nethttp.MethodPost,
			"/api/v1/reportes/texto", body, "")
		if resp.StatusCode != 200 || out["success"] != true {
			t.Fatalf("envío %d: %d %v", i+1, resp.StatusCode, out)
		}
	}
	resp, out := pedirConHeader(t, app, nethttp.MethodPost,
		"/api/v1/reportes/texto", body, "")
	if resp.StatusCode != 429 || out["code"] != "RATE_LIMITED" || resp.Header.Get("Retry-After") == "" {
		t.Fatalf("rate limit: %d %v headers=%v", resp.StatusCode, out, resp.Header)
	}
}

func TestAliasesLegacyYModuloInvalido(t *testing.T) {
	app := newTestApp(t, fakeSender{}, reportes.Config{})
	body := `{"socio":"FUNDABLOCK","base64":"` + jpegBase64(t) + `"}`
	resp, out := pedirConHeader(t, app, nethttp.MethodPost,
		"/api/reportes/enviar-whatsapp", body, "")
	if resp.StatusCode != 200 || out["success"] != true || out["message"] != "Reporte enviado con éxito." {
		t.Fatalf("legacy imagen: %d %v", resp.StatusCode, out)
	}
	resp, out = pedirConHeader(t, app, nethttp.MethodPost,
		"/api/whatsapp/enviar-media", "{", "")
	if resp.StatusCode != 400 || out["success"] != false {
		t.Fatalf("legacy JSON: %d %v", resp.StatusCode, out)
	}

	vacio := nuevoAppModulo(t, &reportes.Module{})
	for _, ruta := range []string{
		"/api/v1/reportes/filtros", "/api/reportes/filtros",
	} {
		resp, _ = pedirConHeader(t, vacio, nethttp.MethodGet, ruta, "", "")
		if resp.StatusCode != 503 {
			t.Fatalf("%s sin servicio: %d", ruta, resp.StatusCode)
		}
	}
	resp, out = pedirConHeader(t, vacio, nethttp.MethodPost,
		"/api/v1/reportes/texto", body, "")
	if resp.StatusCode != 503 || out["code"] != "SERVICE_UNAVAILABLE" {
		t.Fatalf("texto sin servicio: %d %v", resp.StatusCode, out)
	}
}

func TestMapeoDeErroresYResultadoDTO(t *testing.T) {
	casos := []struct {
		err    error
		status int
		code   string
	}{
		{context.Canceled, 504, "TIMEOUT"},
		{context.DeadlineExceeded, 504, "TIMEOUT"},
		{reportes.ErrDestinoRequerido, 400, "DESTINO_REQUERIDO"},
		{reportes.ErrSocioNoEncontrado, 404, "SOCIO_NO_ENCONTRADO"},
		{reportes.ErrJIDNoDisponible, 422, "JID_NO_DISPONIBLE"},
		{reportes.ErrJIDInvalido, 422, "JID_INVALIDO"},
		{reportes.ErrImagenRequerida, 400, "IMAGEN_REQUERIDA"},
		{reportes.ErrImagenMuyGrande, 413, "IMAGEN_MUY_GRANDE"},
		{reportes.ErrImagenFormato, 415, "IMAGEN_FORMATO"},
		{reportes.ErrImagenDimensiones, 422, "IMAGEN_DIMENSIONES"},
		{reportes.ErrTextoRequerido, 400, "TEXTO_REQUERIDO"},
		{reportes.ErrEnvioFallido, 502, "EVOLUTION_UNAVAILABLE"},
		{errors.New("otro"), 500, "INTERNAL_ERROR"},
	}
	for _, caso := range casos {
		got := mapearError(caso.err)
		if got.status != caso.status || got.code != caso.code {
			t.Fatalf("mapearError(%v) = %+v", caso.err, got)
		}
	}
	dto := resultadoDe(reportes.Resultado{JID: "j", MessageID: "m", Estado: "SENT", Simulado: true, Duplicado: true, Bytes: 4})
	if dto.JID != "j" || dto.MessageID != "m" || dto.Status != "SENT" ||
		!dto.Simulated || !dto.Duplicate || dto.Bytes != 4 {
		t.Fatalf("resultadoDe = %+v", dto)
	}
}
