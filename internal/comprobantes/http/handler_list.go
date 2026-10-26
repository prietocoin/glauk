package http

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/comprobantes/domain"
	"glauk/internal/comprobantes/dto"
	"glauk/internal/platform/httpx"
)

// Listar responde GET /api/v1/comprobantes con un array raíz.
//
// Normaliza el orden, valida las fechas y aplica la ventana limit/offset antes
// de llamar al servicio. Los metadatos de paginación viajan en cabeceras para
// conservar el contrato de array raíz que consumen los paneles históricos.
func (d Deps) Listar(c *fiber.Ctx) error {
	c.Set("Cache-Control", "no-store")
	var filters dto.ListFilters
	if err := c.QueryParser(&filters); err != nil {
		return httpx.Fail(c, fiber.StatusBadRequest, "VALIDATION_ERROR",
			"filtros de consulta inválidos")
	}
	if err := dto.NormalizeListFilters(&filters); err != nil {
		return httpx.Fail(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}
	items, total, err := d.Service.List(c.UserContext(), filters)
	if err != nil {
		return responderError(c, err)
	}
	if items == nil {
		items = []domain.Comprobante{}
	}
	escribirPaginacion(c, filters, len(items), total)
	return c.JSON(items)
}

// escribirPaginacion expone la ventana aplicada y el tamaño del universo
// filtrado para que el cliente pueda cargar bloques progresivamente.
func escribirPaginacion(c *fiber.Ctx, filters domain.ListFilters, devueltos, total int) {
	c.Set("X-Total-Count", strconv.Itoa(total))
	c.Set("X-Limit", strconv.Itoa(filters.Limit))
	c.Set("X-Offset", strconv.Itoa(filters.Offset))
	hayMas := filters.Offset+devueltos < total
	c.Set("X-Has-More", strconv.FormatBool(hayMas))
}
