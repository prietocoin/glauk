package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func adminApp(expected string) *fiber.App {
	app := fiber.New()
	app.Use(RequestID())
	app.Get("/admin", RequireAdminKey(expected), func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).SendString("admin")
	})
	return app
}

func TestVerifyAdmin(t *testing.T) {
	if VerifyAdmin("", "") {
		t.Fatal("una clave esperada vacía nunca debe aceptar")
	}
	if VerifyAdmin("clave", "") {
		t.Fatal("una clave esperada vacía nunca debe aceptar")
	}
	if VerifyAdmin("clave", "otra") {
		t.Fatal("clave incorrecta aceptada")
	}
	if VerifyAdmin("clave-larga", "clave") {
		t.Fatal("prefijo aceptado")
	}
	if !VerifyAdmin("clave-segura", "clave-segura") {
		t.Fatal("clave correcta rechazada")
	}
}

func TestRequireAdminKeyDisabled(t *testing.T) {
	response, err := adminApp("").Test(httptest.NewRequest("GET", "/admin", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 503 {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if envelope := decodeEnvelope(t, response); envelope.Code != "ADMIN_DISABLED" {
		t.Fatalf("código = %q", envelope.Code)
	}
}

func TestRequireAdminKeyRejectsInvalid(t *testing.T) {
	request := httptest.NewRequest("GET", "/admin", nil)
	request.Header.Set("X-Admin-Key", "incorrecta")
	response, err := adminApp("correcta").Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 401 {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if envelope := decodeEnvelope(t, response); envelope.Code != "UNAUTHORIZED" {
		t.Fatalf("código = %q", envelope.Code)
	}
}

func TestRequireAdminKeyAcceptsHeaderAndCookie(t *testing.T) {
	request := httptest.NewRequest("GET", "/admin", nil)
	request.Header.Set("X-Admin-Key", "correcta")
	response, err := adminApp("correcta").Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("header status = %d", response.StatusCode)
	}

	request = httptest.NewRequest("GET", "/admin", nil)
	request.AddCookie(&http.Cookie{Name: "admin_key", Value: "correcta"})
	response, err = adminApp("correcta").Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("cookie status = %d", response.StatusCode)
	}
}
