package http

import "github.com/gofiber/fiber/v2"

const aliasSunset = "Sat, 31 Dec 2027 23:59:59 GMT"

func deprecated(successor string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Deprecation", "true")
		c.Set("Sunset", aliasSunset)
		c.Set("Link", "<"+successor+">; rel=\"successor-version\"")
		return c.Next()
	}
}

func guardOrNext(guard fiber.Handler) fiber.Handler {
	if guard != nil {
		return guard
	}
	return func(c *fiber.Ctx) error { return c.Next() }
}
func Register(app *fiber.App, h *Handler, guard fiber.Handler) {
	RegisterCanonical(app, h, guard)
	RegisterAliases(app, h, guard)
}

// RegisterCanonical monta solo /api/v1/reportes/* para respetar §5.12.
func RegisterCanonical(app *fiber.App, h *Handler, guard fiber.Handler) {
	if app == nil || h == nil {
		return
	}
	guard = guardOrNext(guard)
	v1 := app.Group("/api/v1/reportes")
	v1.Get("/filtros", h.Filtros)
	v1.Post("/imagen", guard, h.EnviarImagen)
	v1.Post("/texto", guard, h.EnviarTexto)
}

// RegisterAliases conserva los tres nombres históricos de §3.8.
func RegisterAliases(app *fiber.App, h *Handler, guard fiber.Handler) {
	if app == nil || h == nil {
		return
	}
	guard = guardOrNext(guard)
	app.Get("/api/reportes/filtros", deprecated("/api/v1/reportes/filtros"), h.FiltrosLegacy)
	app.Post("/api/reportes/enviar-whatsapp", deprecated("/api/v1/reportes/imagen"),
		guard, h.EnviarImagenLegacy)
	app.Post("/api/whatsapp/enviar-media", deprecated("/api/v1/reportes/imagen"),
		guard, h.EnviarImagenLegacy)
}
