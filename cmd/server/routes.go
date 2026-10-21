package main

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	comprobanteshttp "glauk/internal/comprobantes/http"
	perfileshttp "glauk/internal/perfiles/http"
	"glauk/internal/platform/httpx"
	reporteshttp "glauk/internal/reportes/http"
	tasashttp "glauk/internal/tasas/http"
)

// registerRoutes sigue el orden normativo de la seccion 5.12. Las rutas
// canonicas se registran antes que los alias; los alias especificos antes que
// los generales; /api/* desconocido se resuelve antes del fallback estatico y
// de la SPA. Las dependencias ya vienen construidas por wire.
func registerRoutes(app *fiber.App, deps *Dependencies) error {
	if app == nil || deps == nil || deps.Config == nil {
		return fmt.Errorf("dependencias de rutas incompletas")
	}

	registerHealth(app, deps)
	if err := registerCanonical(app, deps); err != nil {
		return err
	}
	registerLegacySpecific(app, deps)
	registerLegacyGeneral(app, deps)
	registerAtenea(app, deps)
	registerFallbacks(app, deps)
	return nil
}

// registerHealth expone las sondas antes que cualquier otro grupo. No usan
// autenticacion y deben seguir respondiendo aunque el resto falle; por eso se
// registran primero y no dependen de ningun handler de dominio.
func registerHealth(app *fiber.App, deps *Dependencies) {
	if deps.Health != nil {
		app.Get("/health", deps.Health)
	}
	if deps.Ready != nil {
		app.Get("/ready", deps.Ready)
	}
}

// registerCanonical monta una sola vez la superficie /api/v1 de cada dominio.
// Comprobantes separa sus rutas canonicas de la consola administrativa para
// respetar el orden: reportes antes de admin.
func registerCanonical(app *fiber.App, deps *Dependencies) error {
	if deps.Tasas != nil {
		tasashttp.Register(app, deps.Tasas)
	}
	if err := perfileshttp.Register(app, deps.Perfiles); err != nil {
		return err
	}
	if err := comprobanteshttp.RegisterCanonical(app, deps.Comprobantes); err != nil {
		return err
	}
	reporteshttp.RegisterCanonical(app, deps.Reportes, deps.Guard)
	comprobanteshttp.RegisterAdmin(app, deps.Comprobantes)
	return nil
}

// registerFallbacks instala la frontera final. RegisterStatic ya agrega el 404
// JSON para /api antes de intentar servir archivos; el Use final cubre metodos
// que no son GET/HEAD y no deben caer en la SPA.
func registerFallbacks(app *fiber.App, deps *Dependencies) {
	app.Use(methodNotAllowed(app.GetRoutes(true)))
	httpx.RegisterStatic(app, deps.Config)
	app.Use(globalNotFound)
}

// globalNotFound es la ultima frontera para metodos que no llegan a la SPA.
// Devuelve siempre el envelope canonico con requestId, igual que el 404 de
// /api, de modo que un cliente programatico nunca reciba HTML por accidente.
// No registra rutas: se monta con Use despues de RegisterStatic.
// Su contenido no depende de la configuracion ni del entorno.
func globalNotFound(c *fiber.Ctx) error {
	return httpx.Fail(c, fiber.StatusNotFound, "NOT_FOUND", "recurso no encontrado")
}
