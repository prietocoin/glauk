package ingesta

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/platform/httpx"
)

// EvolutionWebhookPath es la ruta pública que configura Evolution. No usa la
// API key administrativa: Evolution no la envía en los eventos.
const EvolutionWebhookPath = "/api/v1/webhooks/evolution"

// Register monta el webhook. La ruta existe siempre; si INGESTA_ENABLED es
// false responde 503 para que el operador detecte la configuración faltante.
func Register(app *fiber.App, module *Module) error {
	if app == nil {
		return errors.New("ingesta: aplicación Fiber nil")
	}
	app.Post(EvolutionWebhookPath, handler(module))
	return nil
}

func handler(module *Module) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if module == nil || !module.Enabled() {
			return httpx.Fail(c, fiber.StatusServiceUnavailable,
				"INGESTA_DESHABILITADA", "el pipeline de ingesta no está habilitado")
		}
		decision, err := module.Recibir(c.UserContext(), c.Body())
		switch {
		case err == nil:
			return httpx.OKFlat(c, map[string]any{"status": decision.Status})
		case errors.Is(err, ErrInvalidPayload):
			return httpx.Fail(c, fiber.StatusBadRequest, "PAYLOAD_INVALIDO", err.Error())
		case errors.Is(err, ErrQueueFull):
			return httpx.Fail(c, fiber.StatusServiceUnavailable, "COLA_SATURADA", err.Error())
		default:
			return httpx.Fail(c, fiber.StatusInternalServerError, "INGESTA_FALLIDA", err.Error())
		}
	}
}
