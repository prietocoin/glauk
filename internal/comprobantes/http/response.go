package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/comprobantes/domain"
	"glauk/internal/platform/httpx"
)

// ErrServiceNotConfigured se devuelve al montar el router sin servicio.
var ErrServiceNotConfigured = errors.New("comprobantes: servicio no configurado")

func responderError(c *fiber.Ctx, err error) error {
	var validation *domain.ValidationError
	switch {
	case errors.As(err, &validation):
		return httpx.Fail(c, fiber.StatusBadRequest, "VALIDATION_ERROR", validation.Message)
	case errors.Is(err, domain.ErrComprobanteNotFound):
		return httpx.Fail(c, fiber.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrQueueItemNotFound):
		return httpx.Fail(c, fiber.StatusNotFound, "NOT_FOUND", err.Error())
	default:
		return httpx.Fail(c, fiber.StatusInternalServerError, "INTERNAL_ERROR",
			"error interno al procesar comprobantes")
	}
}

func exito(c *fiber.Ctx, message string, data any) error {
	return c.JSON(fiber.Map{"success": true, "message": message, "data": data})
}

func mensaje(c *fiber.Ctx, message string) error {
	return c.JSON(fiber.Map{"success": true, "message": message})
}
