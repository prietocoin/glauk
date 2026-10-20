package http

import (
	"github.com/gofiber/fiber/v2"

	"glauk/internal/comprobantes/domain"
)

// Ultimas responde GET /api/v1/comprobantes/tasas/ultimas.
//
// Devuelve el lote completo, incluido el fallback T001, sin envolverlo en data
// para conservar el contrato que ya consume el panel de auditoría.
func (d Deps) Ultimas(c *fiber.Ctx) error {
	lot, err := d.Service.Latest(c.UserContext())
	if err != nil {
		return responderError(c, err)
	}
	return c.JSON(lot)
}

// Historial responde GET /api/v1/comprobantes/tasas/historial.
//
// La respuesta es siempre un array raíz; nunca null, aunque no haya lotes.
func (d Deps) Historial(c *fiber.Ctx) error {
	lots, err := d.Service.History(c.UserContext())
	if err != nil {
		return responderError(c, err)
	}
	if lots == nil {
		lots = []domain.LoteTasa{}
	}
	return c.JSON(lots)
}
