package http

import (
	"github.com/gofiber/fiber/v2"

	"glauk/internal/comprobantes/dto"
	"glauk/internal/platform/httpx"
)

// Liquidar responde POST /api/v1/comprobantes/liquidar.
func (d Deps) Liquidar(c *fiber.Ctx) error {
	var input dto.LiquidarInput
	if err := c.BodyParser(&input); err != nil {
		return httpx.Fail(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "payload JSON inválido")
	}
	liq, err := d.Service.Liquidar(c.UserContext(), input)
	if err != nil {
		return responderError(c, err)
	}
	return exito(c, "liquidación registrada correctamente", liq)
}

// Actualizar responde PUT /api/v1/comprobantes/:hashLargo.
func (d Deps) Actualizar(c *fiber.Ctx) error {
	var input dto.UpdateInput
	if err := c.BodyParser(&input); err != nil {
		return httpx.Fail(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "payload JSON inválido")
	}
	liq, err := d.Service.Update(c.UserContext(), c.Params("hashLargo"), input)
	if err != nil {
		return responderError(c, err)
	}
	return exito(c, "comprobante actualizado correctamente", liq)
}

// Eliminar responde DELETE /api/v1/comprobantes/:hashLargo.
func (d Deps) Eliminar(c *fiber.Ctx) error {
	if err := d.Service.Delete(c.UserContext(), c.Params("hashLargo")); err != nil {
		return responderError(c, err)
	}
	return mensaje(c, "comprobante eliminado correctamente")
}

// Releer responde POST /api/v1/comprobantes/:hashLargo/releer.
func (d Deps) Releer(c *fiber.Ctx) error {
	if err := d.Service.Releer(c.UserContext(), c.Params("hashLargo")); err != nil {
		return responderError(c, err)
	}
	return mensaje(c, "re-lectura IA marcada para reprocesamiento")
}
