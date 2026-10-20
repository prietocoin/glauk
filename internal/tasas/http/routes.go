package http

import "github.com/gofiber/fiber/v2"

// Register monta exactamente las trece rutas canónicas de Tasas bajo /api/v1.
// /disparar-lote comparte handler con /publicar-lote porque en el servicio Node
// eran alias; /render atiende GET y POST con la misma lógica.
//
// Los alias históricos /api/tasas/* no se registran aquí: cmd/server los monta
// reutilizando estos mismos handlers, de modo que el dominio tenga una sola
// definición de cada contrato.

func Register(app *fiber.App, deps *Deps) {
	grupo := app.Group("/api/v1/tasas")

	grupo.Get("/monitor", deps.Monitor)
	grupo.Get("/socios", deps.Socios)
	grupo.Get("/hoo-borrador", deps.BorradorHoo)

	grupo.Put("/socios/:socio", deps.GuardarSocio)
	grupo.Post("/guardar-lote", deps.GuardarLote)
	grupo.Post("/publicar-lote", deps.PublicarLote)

	grupo.Post("/notificar-remesas", deps.NotificarRemesas)

	grupo.Get("/calcular/:socio", deps.Calcular)
	grupo.Get("/render/:socio", deps.Render)
	grupo.Post("/render", deps.Render)
	grupo.Post("/disparar", deps.Disparar)
	grupo.Post("/disparar-lote", deps.PublicarLote)
	grupo.Delete("/lote/:id_tasa", deps.EliminarLote)
}
