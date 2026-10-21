package main

import (
	"sync/atomic"

	"github.com/gofiber/fiber/v2"

	reporteshttp "glauk/internal/reportes/http"
)

const sunsetAlias = "Sat, 31 Dec 2027 23:59:59 GMT"

var aliasUsage atomic.Int64

// Deprecated marca un alias con las cabeceras de §5.7.2. El contador global
// permite observar su uso sin acoplar los handlers a métricas externas.
func Deprecated(successor string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		aliasUsage.Add(1)
		c.Set("Deprecation", "true")
		c.Set("Sunset", sunsetAlias)
		c.Set("Link", "<"+successor+">; rel=\"successor-version\"")
		return c.Next()
	}
}

func alias(app *fiber.App, method, path, successor string, handlers ...fiber.Handler) {
	chain := make([]fiber.Handler, 0, len(handlers)+1)
	chain = append(chain, Deprecated(successor))
	chain = append(chain, handlers...)
	app.Add(method, path, chain...)
}

// registerLegacySpecific monta los alias que no colisionan con un prefijo
// general. /api/tasas/ultimas e /historial pertenecen a Comprobantes.
func registerLegacySpecific(app *fiber.App, deps *Dependencies) {
	reporteshttp.RegisterAliases(app, deps.Reportes, deps.Guard)
	alias(app, fiber.MethodGet, "/api/tasas/ultimas",
		"/api/v1/comprobantes/tasas/ultimas", deps.Comprobantes.Ultimas)
	alias(app, fiber.MethodGet, "/api/tasas/historial",
		"/api/v1/comprobantes/tasas/historial", deps.Comprobantes.Historial)

	if deps.Tasas == nil {
		return
	}
	t := deps.Tasas
	alias(app, fiber.MethodGet, "/api/tasas/monitor", "/api/v1/tasas/monitor", t.Monitor)
	alias(app, fiber.MethodGet, "/api/tasas/socios", "/api/v1/tasas/socios", t.Socios)
	alias(app, fiber.MethodGet, "/api/tasas/hoo-borrador", "/api/v1/tasas/hoo-borrador", t.BorradorHoo)
	alias(app, fiber.MethodPut, "/api/tasas/socios/:socio", "/api/v1/tasas/socios/:socio", t.GuardarSocio)
	alias(app, fiber.MethodPost, "/api/tasas/guardar-lote", "/api/v1/tasas/guardar-lote", t.GuardarLote)
	alias(app, fiber.MethodPost, "/api/tasas/publicar-lote", "/api/v1/tasas/publicar-lote", t.PublicarLote)
	alias(app, fiber.MethodPost, "/api/tasas/notificar-remesas", "/api/v1/tasas/notificar-remesas", t.NotificarRemesas)
	alias(app, fiber.MethodGet, "/api/tasas/calcular/:socio", "/api/v1/tasas/calcular/:socio", t.Calcular)
	alias(app, fiber.MethodGet, "/api/tasas/render/:socio", "/api/v1/tasas/render/:socio", t.Render)
	alias(app, fiber.MethodPost, "/api/tasas/render", "/api/v1/tasas/render", t.Render)
	alias(app, fiber.MethodPost, "/api/tasas/disparar", "/api/v1/tasas/disparar", t.Disparar)
	alias(app, fiber.MethodPost, "/api/tasas/disparar-lote", "/api/v1/tasas/disparar-lote", t.PublicarLote)
	alias(app, fiber.MethodDelete, "/api/tasas/lote/:id_tasa", "/api/v1/tasas/lote/:id_tasa", t.EliminarLote)
	alias(app, fiber.MethodGet, "/api/tasas/fetch-hoo", "/api/v1/tasas/hoo-borrador", t.BorradorHoo)
	alias(app, fiber.MethodPost, "/api/tasas/publicar", "/api/v1/tasas/publicar-lote", publicarAlias(t))
	alias(app, fiber.MethodPost, "/api/tasas/reenviar", "/api/v1/tasas/publicar-lote", reenviarAlias(t))
	alias(app, fiber.MethodGet, "/api/preview-image/:identificador", "/api/v1/tasas/render/:socio", previewImage(t))
	alias(app, fiber.MethodGet, "/api/preview-image", "/api/v1/tasas/render/:socio", previewImage(t))
}

// registerLegacyGeneral monta los alias de prefijo amplio. /api/reportes es el
// listado de Comprobantes y se registra después de /api/reportes/filtros.
func registerLegacyGeneral(app *fiber.App, deps *Dependencies) {
	p := deps.Perfiles
	alias(app, fiber.MethodGet, "/api/directorio", "/api/v1/perfiles", p.Listar)
	alias(app, fiber.MethodDelete, "/api/directorio/:nombre", "/api/v1/perfiles/:nombre", p.Eliminar)
	alias(app, fiber.MethodGet, "/api/socios", "/api/v1/perfiles", p.Listar)
	alias(app, fiber.MethodPost, "/api/socios/config", "/api/v1/perfiles/guardar", legacyGuardarPerfil(p))
	alias(app, fiber.MethodPatch, "/api/socios/:nombre/estado", "/api/v1/perfiles/:nombre/estado", p.Toggle)
	alias(app, fiber.MethodPatch, "/api/socios/desactivar-todos", "/api/v1/perfiles/apagar-todos", p.ApagarTodos)
	alias(app, fiber.MethodPost, "/api/socios/guardar-vigentes", "/api/v1/perfiles/plantilla/memorizar", p.Memorizar)
	alias(app, fiber.MethodPost, "/api/socios/restaurar-vigentes", "/api/v1/perfiles/plantilla/restaurar", p.Restaurar)

	c := deps.Comprobantes
	alias(app, fiber.MethodGet, "/api/comprobantes", "/api/v1/comprobantes", c.Listar)
	alias(app, fiber.MethodGet, "/api/cola", "/api/v1/comprobantes", c.Listar)
	alias(app, fiber.MethodGet, "/api/reportes", "/api/v1/comprobantes", c.Listar)
	alias(app, fiber.MethodGet, "/api/reportes/operaciones", "/api/v1/comprobantes", c.Listar)
	alias(app, fiber.MethodPost, "/api/comprobantes/liquidar", "/api/v1/comprobantes/liquidar", c.Liquidar)
	alias(app, fiber.MethodPost, "/api/comprobantes/:hashLargo/releer", "/api/v1/comprobantes/:hashLargo/releer", c.Releer)
	alias(app, fiber.MethodPut, "/api/comprobantes/:hashLargo", "/api/v1/comprobantes/:hashLargo", c.Actualizar)
	alias(app, fiber.MethodDelete, "/api/comprobantes/:hashLargo", "/api/v1/comprobantes/:hashLargo", c.Eliminar)
}
