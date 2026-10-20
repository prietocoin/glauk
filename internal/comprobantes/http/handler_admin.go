package http

import (
	"github.com/gofiber/fiber/v2"

	"glauk/internal/comprobantes/dto"
	"glauk/internal/platform/httpx"
)

// Cola responde GET /api/v1/admin/cola.
func (d Deps) Cola(c *fiber.Ctx) error {
	items, err := d.Service.Queue(c.UserContext())
	if err != nil {
		return responderError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "data": items})
}

// ActualizarCola responde PUT /api/v1/admin/cola/:hashLargo.
func (d Deps) ActualizarCola(c *fiber.Ctx) error {
	var input dto.UpdateInput
	if err := c.BodyParser(&input); err != nil {
		return httpx.Fail(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "payload JSON inválido")
	}
	data, err := d.Service.UpdateQueue(c.UserContext(), c.Params("hashLargo"), input)
	if err != nil {
		return responderError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "data": data})
}

// EliminarCola responde DELETE /api/v1/admin/cola/:hashLargo.
func (d Deps) EliminarCola(c *fiber.Ctx) error {
	if err := d.Service.DeleteQueue(c.UserContext(), c.Params("hashLargo")); err != nil {
		return responderError(c, err)
	}
	return mensaje(c, "registro eliminado correctamente")
}
