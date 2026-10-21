package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/platform/httpx"
)

// Register monta las diez rutas canónicas de Comprobantes.
func Register(app *fiber.App, deps Deps) error {
	if err := RegisterCanonical(app, deps); err != nil {
		return err
	}
	RegisterAdmin(app, deps)
	return nil
}

// RegisterCanonical monta solo las siete rutas de §5.5.1. cmd/server lo usa
// para respetar el orden canónico antes de la consola administrativa.
func RegisterCanonical(app *fiber.App, deps Deps) error {
	if err := deps.Valida(); err != nil {
		return err
	}
	app.Get(BasePath, deps.Listar)
	app.Post(BasePath+"/liquidar", deps.Liquidar)
	app.Post(BasePath+"/:hashLargo/releer", deps.Releer)
	app.Put(BasePath+"/:hashLargo", deps.Actualizar)
	app.Delete(BasePath+"/:hashLargo", deps.Eliminar)
	app.Get(BasePath+"/tasas/ultimas", deps.Ultimas)
	app.Get(BasePath+"/tasas/historial", deps.Historial)
	return nil
}

// RegisterAdmin monta la consola de §5.8 con su propia clave.
func RegisterAdmin(app *fiber.App, deps Deps) {
	admin := app.Group("/api/v1/admin", requireAdmin(deps.AdminKey))
	admin.Get("/cola", deps.Cola)
	admin.Put("/cola/:hashLargo", deps.ActualizarCola)
	admin.Delete("/cola/:hashLargo", deps.EliminarCola)
}

func requireAdmin(expected string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		if strings.TrimSpace(expected) == "" {
			return httpx.Fail(c, fiber.StatusServiceUnavailable, "ADMIN_DISABLED",
				"la consola administrativa no está configurada")
		}
		if !httpx.VerifyAdmin(c.Get("X-Admin-Key"), expected) {
			return httpx.Fail(c, fiber.StatusUnauthorized, "UNAUTHORIZED",
				"clave administrativa inválida")
		}
		return c.Next()
	}
}
