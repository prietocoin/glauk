package httpx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func errorApp(err error) (*fiber.App, *bytes.Buffer) {
	buffer := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buffer, nil))
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler(logger)})
	app.Use(RequestID())
	app.Get("/", func(c *fiber.Ctx) error { return err })
	return app, buffer
}

func TestErrorHandlerMapsHTTPError(t *testing.T) {
	cause := errors.New("detalle privado")
	app, _ := errorApp(fmt.Errorf("contexto: %w", &HTTPError{
		Status: 422, Code: "VALIDATION_ERROR", Message: "campo inválido", Err: cause,
	}))
	response, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 422 {
		t.Fatalf("status = %d", response.StatusCode)
	}
	envelope := decodeEnvelope(t, response)
	if envelope.Code != "VALIDATION_ERROR" || envelope.Error != "campo inválido" {
		t.Fatalf("envelope inesperado: %+v", envelope)
	}
	if strings.Contains(responseBody(t, response), "detalle privado") {
		t.Fatal("se filtró la causa interna")
	}
}

func TestErrorHandlerMapsFiberError(t *testing.T) {
	app, _ := errorApp(fiber.ErrNotFound)
	response, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 404 {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if envelope := decodeEnvelope(t, response); envelope.Code != "NOT_FOUND" {
		t.Fatalf("código = %q", envelope.Code)
	}
}

func TestErrorHandlerMapsDeadlineToTimeout(t *testing.T) {
	app, _ := errorApp(fmt.Errorf("consulta: %w", context.DeadlineExceeded))
	response, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 504 {
		t.Fatalf("status = %d", response.StatusCode)
	}
	envelope := decodeEnvelope(t, response)
	if envelope.Code != "TIMEOUT" || envelope.Error == "" {
		t.Fatalf("envelope inesperado: %+v", envelope)
	}
}

func TestErrorHandlerHidesUnknownCause(t *testing.T) {
	app, logs := errorApp(errors.New("password=supersecreto ruta=C:/privado"))
	response, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 500 {
		t.Fatalf("status = %d", response.StatusCode)
	}
	envelope := decodeEnvelope(t, response)
	if envelope.Code != "INTERNAL" || envelope.Error != "error interno del servidor" {
		t.Fatalf("envelope inesperado: %+v", envelope)
	}
	if strings.Contains(responseBody(t, response), "supersecreto") {
		t.Fatal("el 500 filtró el mensaje interno")
	}
	if !strings.Contains(logs.String(), "supersecreto") {
		t.Fatal("la causa no se registró para diagnóstico")
	}
}

func TestHTTPErrorStableDefaults(t *testing.T) {
	app, _ := errorApp(&HTTPError{Status: 500, Err: errors.New("secreto")})
	response, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	envelope := decodeEnvelope(t, response)
	if response.StatusCode != 500 || envelope.Code != "INTERNAL" ||
		envelope.Error != "error interno del servidor" {
		t.Fatalf("respuesta inesperada: %d %+v", response.StatusCode, envelope)
	}
}
