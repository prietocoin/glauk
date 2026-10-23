package main

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	comprobanteshttp "glauk/internal/comprobantes/http"
	"glauk/internal/ingesta"
	perfileshttp "glauk/internal/perfiles/http"
	"glauk/internal/platform/httpx"
	reporteshttp "glauk/internal/reportes/http"
	tasashttp "glauk/internal/tasas/http"
)

// registerRoutes registra la API completa. Las rutas canonicas van antes que
// los alias y estos antes que las rutas generales. Las dependencias ya vienen
// construidas por wire.
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
	registerFallbacks(app)
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
	if err := ingesta.Register(app, deps.Ingesta); err != nil {
		return err
	}
	return nil
}

// registerFallbacks instala la frontera final: 405 para rutas conocidas con
// metodo incorrecto y 404 JSON para todo lo demas.
func registerFallbacks(app *fiber.App) {
	app.Use(methodNotAllowed(app.GetRoutes(true)))
	app.Use(globalNotFound)
}

// globalNotFound es la ultima frontera de la API. Devuelve siempre el envelope
// canonico con requestId, de modo que un cliente programatico nunca reciba
// HTML por accidente. No registra rutas: se monta con Use.
// Su contenido no depende de la configuracion ni del entorno.
func globalNotFound(c *fiber.Ctx) error {
	return httpx.Fail(c, fiber.StatusNotFound, "NOT_FOUND", "recurso no encontrado")
}
