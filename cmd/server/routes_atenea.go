package main

import "github.com/gofiber/fiber/v2"

// registerAtenea conserva los alias heredados. Las rutas específicas se
// registran antes que /api/v1/atenea/reportes para no cambiar su semántica.
func registerAtenea(app *fiber.App, deps *Dependencies) {
	registerAteneaReportes(app, deps)
	registerAteneaPerfiles(app, deps)
	registerAteneaComprobantes(app, deps)
}

func registerAteneaReportes(app *fiber.App, deps *Dependencies) {
	h := deps.Reportes
	if h == nil {
		return
	}
	guard := deps.Guard
	if guard == nil {
		guard = func(c *fiber.Ctx) error { return c.Next() }
	}
	alias(app, fiber.MethodGet, "/api/v1/atenea/reportes/filtros",
		"/api/v1/reportes/filtros", h.FiltrosLegacy)
	alias(app, fiber.MethodPost, "/api/v1/atenea/reportes/enviar-whatsapp",
		"/api/v1/reportes/imagen", guard, h.EnviarImagenLegacy)
	alias(app, fiber.MethodPost, "/api/v1/atenea/whatsapp/enviar-media",
		"/api/v1/reportes/imagen", guard, h.EnviarImagenLegacy)
}

func registerAteneaPerfiles(app *fiber.App, deps *Dependencies) {
	p := deps.Perfiles
	alias(app, fiber.MethodGet, "/api/v1/atenea/directorio", "/api/v1/perfiles", p.Listar)
	alias(app, fiber.MethodDelete, "/api/v1/atenea/directorio/:nombre", "/api/v1/perfiles/:nombre", p.Eliminar)
	alias(app, fiber.MethodGet, "/api/v1/atenea/socios", "/api/v1/perfiles", p.Listar)
	alias(app, fiber.MethodPost, "/api/v1/atenea/socios/config", "/api/v1/perfiles/guardar", legacyGuardarPerfil(p))
	alias(app, fiber.MethodPatch, "/api/v1/atenea/socios/:nombre/estado", "/api/v1/perfiles/:nombre/estado", p.Toggle)
	alias(app, fiber.MethodPatch, "/api/v1/atenea/socios/desactivar-todos", "/api/v1/perfiles/apagar-todos", p.ApagarTodos)
	alias(app, fiber.MethodPost, "/api/v1/atenea/socios/guardar-vigentes", "/api/v1/perfiles/plantilla/memorizar", p.Memorizar)
	alias(app, fiber.MethodPost, "/api/v1/atenea/socios/restaurar-vigentes", "/api/v1/perfiles/plantilla/restaurar", p.Restaurar)
}

func registerAteneaComprobantes(app *fiber.App, deps *Dependencies) {
	c := deps.Comprobantes
	alias(app, fiber.MethodGet, "/api/v1/atenea/cola", "/api/v1/comprobantes", c.Listar)
	alias(app, fiber.MethodGet, "/api/v1/atenea/reportes", "/api/v1/comprobantes", c.Listar)
	alias(app, fiber.MethodGet, "/api/v1/atenea/reportes/operaciones", "/api/v1/comprobantes", c.Listar)
	alias(app, fiber.MethodGet, "/api/v1/atenea/tasas/ultimas", "/api/v1/comprobantes/tasas/ultimas", c.Ultimas)
	alias(app, fiber.MethodGet, "/api/v1/atenea/tasas/historial", "/api/v1/comprobantes/tasas/historial", c.Historial)
	alias(app, fiber.MethodGet, "/api/v1/atenea/comprobantes", "/api/v1/comprobantes", c.Listar)
	alias(app, fiber.MethodPost, "/api/v1/atenea/comprobantes/liquidar", "/api/v1/comprobantes/liquidar", c.Liquidar)
	alias(app, fiber.MethodPost, "/api/v1/atenea/comprobantes/:hashLargo/releer", "/api/v1/comprobantes/:hashLargo/releer", c.Releer)
	alias(app, fiber.MethodPut, "/api/v1/atenea/comprobantes/:hashLargo", "/api/v1/comprobantes/:hashLargo", c.Actualizar)
	alias(app, fiber.MethodDelete, "/api/v1/atenea/comprobantes/:hashLargo", "/api/v1/comprobantes/:hashLargo", c.Eliminar)
}
