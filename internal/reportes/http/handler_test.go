package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/reportes"
)

type fakeSocios struct{}

func (fakeSocios) SocioPorNombre(_ context.Context, nombre string) (reportes.Socio, error) {
	if strings.EqualFold(nombre, "NADIE") {
		return reportes.Socio{}, reportes.ErrSocioNoEncontrado
	}
	return reportes.Socio{Nombre: nombre, IDGrupo: "120363012345678901@g.us"}, nil
}

func (fakeSocios) Socios(context.Context, string) ([]reportes.Socio, error) {
	return []reportes.Socio{}, nil
}

type fakeMovimientos struct{}

func (fakeMovimientos) Filtros(context.Context, string) (reportes.Filtros, error) {
	return reportes.Filtros{Roles: []string{"ADMIN"}, Entidades: []string{"GENERAL"}}, nil
}

func (fakeMovimientos) Movimientos(context.Context, reportes.ConsultaMovimientos) ([]reportes.Movimiento, error) {
	return []reportes.Movimiento{}, nil
}

func (fakeMovimientos) Resumen(context.Context, reportes.ConsultaMovimientos) (reportes.Resumen, error) {
	return reportes.Resumen{}, nil
}

type fakeSender struct{ err error }

func (f fakeSender) EnviarImagen(context.Context, reportes.MensajeImagen) (reportes.Recibo, error) {
	if f.err != nil {
		return reportes.Recibo{}, f.err
	}
	return reportes.Recibo{MessageID: "MSG-1", Estado: "SENT", Simulado: true}, nil
}

func (f fakeSender) EnviarTexto(context.Context, reportes.MensajeTexto) (reportes.Recibo, error) {
	if f.err != nil {
		return reportes.Recibo{}, f.err
	}
	return reportes.Recibo{MessageID: "MSG-2", Estado: "SENT", Simulado: true}, nil
}

func newTestApp(t *testing.T, sender reportes.WhatsAppSender, cfg reportes.Config) *fiber.App {
	t.Helper()
	service := reportes.NewService(fakeSocios{}, fakeMovimientos{}, sender, cfg)
	app := fiber.New()
	Register(app, NewHandler(&reportes.Module{Service: service}), nil)
	return app
}

func pedir(t *testing.T, app *fiber.App, method, path, body string) (*nethttp.Response, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
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

func jpegBase64(t *testing.T) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestErroresDeContrato(t *testing.T) {
	app := newTestApp(t, fakeSender{}, reportes.Config{})
	valid := jpegBase64(t)
	casos := []struct {
		name   string
		path   string
		body   string
		status int
		code   string
	}{
		{"json inválido", "/api/v1/reportes/imagen", "{", 400, "INVALID_JSON"},
		{"destino", "/api/v1/reportes/imagen", `{"base64":"` + valid + `"}`, 400, "DESTINO_REQUERIDO"},
		{"socio", "/api/v1/reportes/imagen", `{"socio":"NADIE","base64":"` + valid + `"}`, 404, "SOCIO_NO_ENCONTRADO"},
		{"jid", "/api/v1/reportes/imagen", `{"remoteJid":"invalido","base64":"` + valid + `"}`, 422, "JID_INVALIDO"},
		{"formato", "/api/v1/reportes/imagen", `{"socio":"FUNDABLOCK","base64":"bm90IGpwZWc="}`, 415, "IMAGEN_FORMATO"},
		{"texto", "/api/v1/reportes/texto", `{"socio":"FUNDABLOCK"}`, 400, "TEXTO_REQUERIDO"},
	}
	for _, caso := range casos {
		t.Run(caso.name, func(t *testing.T) {
			resp, out := pedir(t, app, nethttp.MethodPost, caso.path, caso.body)
			if resp.StatusCode != caso.status || out["code"] != caso.code {
				t.Fatalf("status=%d code=%v body=%v", resp.StatusCode, out["code"], out)
			}
		})
	}
}

func TestImagenMuyGrandeYEvolutionCaido(t *testing.T) {
	valid := jpegBase64(t)
	pequena := newTestApp(t, fakeSender{}, reportes.Config{ImageMaxBytes: 128})
	resp, out := pedir(t, pequena, nethttp.MethodPost, "/api/v1/reportes/imagen",
		`{"socio":"FUNDABLOCK","base64":"`+valid+`"}`)
	if resp.StatusCode != 413 || out["code"] != "IMAGEN_MUY_GRANDE" {
		t.Fatalf("413: status=%d body=%v", resp.StatusCode, out)
	}

	caida := newTestApp(t, fakeSender{err: errors.New("evolution caído")}, reportes.Config{})
	resp, out = pedir(t, caida, nethttp.MethodPost, "/api/v1/reportes/imagen",
		`{"socio":"FUNDABLOCK","base64":"`+valid+`"}`)
	if resp.StatusCode != 502 || out["code"] != "EVOLUTION_UNAVAILABLE" {
		t.Fatalf("502: status=%d body=%v", resp.StatusCode, out)
	}
}

func TestFiltrosCanonicoYLegacy(t *testing.T) {
	app := newTestApp(t, fakeSender{}, reportes.Config{})
	resp, out := pedir(t, app, nethttp.MethodGet, "/api/v1/reportes/filtros", "")
	if resp.StatusCode != 200 || out["success"] != true {
		t.Fatalf("canónico: %d %v", resp.StatusCode, out)
	}
	data, _ := out["data"].(map[string]any)
	if data == nil || data["hashes"] == nil {
		t.Fatalf("data inválida: %v", out)
	}

	resp, out = pedir(t, app, nethttp.MethodGet, "/api/reportes/filtros", "")
	if resp.StatusCode != 200 || out["success"] != true || out["roles"] == nil {
		t.Fatalf("legacy: %d %v", resp.StatusCode, out)
	}
}
