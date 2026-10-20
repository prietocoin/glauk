package http

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

// Memorizar responde POST /api/v1/perfiles/plantilla/memorizar. Congela el
// estado actual de mostrar en recordar_mostrar; no activa ni apaga socios.
func (d Deps) Memorizar(c *fiber.Ctx) error {
	return d.ejecutarPlantilla(c, "plantilla memorizada", d.Perfiles.Memorizar)
}

// Restaurar responde POST /api/v1/perfiles/plantilla/restaurar. Recupera la
// última plantilla y mantiene GENERAL activo, igual que el flujo histórico.
func (d Deps) Restaurar(c *fiber.Ctx) error {
	return d.ejecutarPlantilla(c, "plantilla restaurada", d.Perfiles.Restaurar)
}

// ejecutarPlantilla comparte el manejo de errores de las dos operaciones.
// Ninguna de las dos recibe body: el payload no puede alterar recordar_mostrar
// ni elegir un subconjunto de socios. Si el repositorio falla, el error se traduce
// con responderError y no se confirmar la operación.
func (d Deps) ejecutarPlantilla(c *fiber.Ctx, texto string, operacion func(context.Context) error) error {
	if err := operacion(c.UserContext()); err != nil {
		return responderError(c, err)
	}
	return mensaje(c, texto)
}
