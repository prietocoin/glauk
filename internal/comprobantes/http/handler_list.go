package http

import (
	"github.com/gofiber/fiber/v2"

	"glauk/internal/comprobantes/domain"
	"glauk/internal/comprobantes/dto"
	"glauk/internal/platform/httpx"
)

// Listar responde GET /api/v1/comprobantes con un array raíz.
//
// Normaliza el orden y valida las fechas antes de llamar al servicio. Un
// formato inválido se traduce a 400 sin consultar PostgreSQL ni proyectar
// filas; un resultado vacío se serializa como [] y nunca como null.
func (d Deps) Listar(c *fiber.Ctx) error {
	var filters dto.ListFilters
	if err := c.QueryParser(&filters); err != nil {
		return httpx.Fail(c, fiber.StatusBadRequest, "VALIDATION_ERROR",
			"filtros de consulta inválidos")
	}
	if err := dto.NormalizeListFilters(&filters); err != nil {
		return httpx.Fail(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}
	items, err := d.Service.List(c.UserContext(), filters)
	if err != nil {
		return responderError(c, err)
	}
	if items == nil {
		items = []domain.Comprobante{}
	}
	return c.JSON(items)
}
