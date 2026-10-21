package main

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/platform/httpx"
)

// registerLegacyAdmin conserva la consola administrativa anterior a /api/v1.
// Los alias comparten el Deprecated global para que su uso quede contado.
func registerLegacyAdmin(app *fiber.App, deps *Dependencies) {
	c := deps.Comprobantes
	alias(app, fiber.MethodGet, "/api/admin/cola", "/api/v1/admin/cola",
		adminAliasGuard(c.AdminKey, false), c.Cola)
	alias(app, fiber.MethodGet, "/api/v1/atenea/admin/cola", "/api/v1/admin/cola",
		adminAliasGuard(c.AdminKey, false), c.Cola)
	alias(app, fiber.MethodPut, "/api/admin/cola/:hashLargo", "/api/v1/admin/cola/:hashLargo",
		adminAliasGuard(c.AdminKey, false), c.ActualizarCola)
	alias(app, fiber.MethodDelete, "/api/admin/cola/:hashLargo", "/api/v1/admin/cola/:hashLargo",
		adminAliasGuard(c.AdminKey, true), c.EliminarCola)
}

// adminAliasGuard replica el contrato de la consola canónica y, solo en el
// DELETE legacy, acepta además {claveAdmin} en el cuerpo.
func adminAliasGuard(expected string, acceptBody bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		if strings.TrimSpace(expected) == "" {
			return httpx.Fail(c, fiber.StatusServiceUnavailable, "ADMIN_DISABLED",
				"la consola administrativa no está configurada")
		}
		provided := c.Get("X-Admin-Key")
		if provided == "" && acceptBody {
			var body struct {
				ClaveAdmin string `json:"claveAdmin"`
			}
			_ = c.BodyParser(&body)
			provided = body.ClaveAdmin
		}
		if !httpx.VerifyAdmin(provided, expected) {
			return httpx.Fail(c, fiber.StatusUnauthorized, "UNAUTHORIZED",
				"clave administrativa inválida")
		}
		return c.Next()
	}
}
