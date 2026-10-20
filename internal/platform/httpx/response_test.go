package httpx

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func responseApp() *fiber.App {
	app := fiber.New()
	app.Use(RequestID())
	app.Get("/ok", func(c *fiber.Ctx) error { return OK(c, fiber.Map{"value": 1}) })
	app.Post("/created", func(c *fiber.Ctx) error { return Created(c, fiber.Map{"id": "x"}) })
	app.Get("/fail", func(c *fiber.Ctx) error {
		return Fail(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "dato inválido")
	})
	app.Get("/fail-code", func(c *fiber.Ctx) error {
		return FailCode(c, fiber.StatusConflict, "CONFLICT", "conflicto")
	})
	app.Get("/flat", func(c *fiber.Ctx) error {
		return OKFlat(c, map[string]any{"total": 2, "nombre": "Glauk"})
	})
	app.Get("/id", func(c *fiber.Ctx) error { return c.SendString(RequestIDFrom(c)) })
	return app
}

func decodeEnvelope(t *testing.T, response *http.Response) Envelope {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	response.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var envelope Envelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("JSON inválido: %v (%s)", err, body)
	}
	return envelope
}

func TestRequestIDValidatesAndGenerates(t *testing.T) {
	app := responseApp()
	for _, valid := range []string{
		"9f2c1e7a-4b8d-4c1e-9f2c-1e7a4b8d4c1e",
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
	} {
		request := httptest.NewRequest("GET", "/id", nil)
		request.Header.Set("X-Request-Id", valid)
		response, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		if body := responseBody(t, response); body != valid {
			t.Fatalf("ID válido reemplazado: %q", body)
		}
	}

	for _, invalid := range []string{"", "no-es-un-id", "1234"} {
		request := httptest.NewRequest("GET", "/id", nil)
		if invalid != "" {
			request.Header.Set("X-Request-Id", invalid)
		}
		response, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		body := responseBody(t, response)
		if len(body) != 32 {
			t.Fatalf("ID generado con longitud %d: %q", len(body), body)
		}
		for i := 0; i < len(body); i++ {
			if !isHex(body[i]) {
				t.Fatalf("ID generado no hexadecimal: %q", body)
			}
		}
		if response.Header.Get("X-Request-Id") != body {
			t.Fatal("el header de respuesta no coincide")
		}
	}
}

func TestOKAndCreatedInjectRequestID(t *testing.T) {
	app := responseApp()
	response, err := app.Test(httptest.NewRequest("GET", "/ok", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("OK status = %d", response.StatusCode)
	}
	envelope := decodeEnvelope(t, response)
	if !envelope.Success || envelope.RequestID == "" {
		t.Fatalf("envelope inesperado: %+v", envelope)
	}
	if response.Header.Get("X-Request-Id") != envelope.RequestID {
		t.Fatal("el header no coincide con requestId")
	}

	response, err = app.Test(httptest.NewRequest("POST", "/created", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 201 {
		t.Fatalf("Created status = %d", response.StatusCode)
	}
	if envelope := decodeEnvelope(t, response); !envelope.Success || envelope.RequestID == "" {
		t.Fatalf("envelope inesperado: %+v", envelope)
	}
}

func TestFailAndFailCodeUseCanonicalError(t *testing.T) {
	app := responseApp()
	response, err := app.Test(httptest.NewRequest("GET", "/fail", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 422 {
		t.Fatalf("Fail status = %d", response.StatusCode)
	}
	envelope := decodeEnvelope(t, response)
	if envelope.Success || envelope.Code != "VALIDATION_ERROR" ||
		envelope.Error != "dato inválido" || envelope.RequestID == "" {
		t.Fatalf("envelope inesperado: %+v", envelope)
	}

	response, err = app.Test(httptest.NewRequest("GET", "/fail-code", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 409 {
		t.Fatalf("FailCode status = %d", response.StatusCode)
	}
	if envelope := decodeEnvelope(t, response); envelope.Code != "CONFLICT" {
		t.Fatalf("código inesperado: %+v", envelope)
	}
}

func TestOKFlatPreservesRootFields(t *testing.T) {
	response, err := responseApp().Test(httptest.NewRequest("GET", "/flat", nil))
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(responseBody(t, response)), &body); err != nil {
		t.Fatal(err)
	}
	if body["success"] != true || body["total"] != float64(2) || body["nombre"] != "Glauk" {
		t.Fatalf("campos planos perdidos: %#v", body)
	}
	if body["requestId"] == "" {
		t.Fatal("falta requestId")
	}
}
