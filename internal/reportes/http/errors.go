package http

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/platform/httpx"
	"glauk/internal/reportes"
)

type errorHTTP struct {
	status  int
	code    string
	message string
}

func mapearError(err error) errorHTTP {
	switch {
	case errors.Is(err, context.DeadlineExceeded),
		errors.Is(err, context.Canceled):
		return errorHTTP{fiber.StatusGatewayTimeout, "TIMEOUT",
			"la operación excedió el tiempo máximo"}
	case errors.Is(err, reportes.ErrDestinoRequerido):
		return errorHTTP{fiber.StatusBadRequest, "DESTINO_REQUERIDO",
			"debe indicar socio o remoteJid"}
	case errors.Is(err, reportes.ErrSocioNoEncontrado):
		return errorHTTP{fiber.StatusNotFound, "SOCIO_NO_ENCONTRADO",
			"socio no encontrado"}
	case errors.Is(err, reportes.ErrJIDNoDisponible):
		return errorHTTP{fiber.StatusUnprocessableEntity, "JID_NO_DISPONIBLE",
			"el socio no tiene un JID configurado"}
	case errors.Is(err, reportes.ErrJIDInvalido):
		return errorHTTP{fiber.StatusUnprocessableEntity, "JID_INVALIDO",
			"el JID no tiene un formato de WhatsApp válido"}
	case errors.Is(err, reportes.ErrImagenRequerida):
		return errorHTTP{fiber.StatusBadRequest, "IMAGEN_REQUERIDA",
			"base64 es obligatorio"}
	case errors.Is(err, reportes.ErrImagenMuyGrande):
		return errorHTTP{fiber.StatusRequestEntityTooLarge, "IMAGEN_MUY_GRANDE",
			"la imagen excede el tamaño máximo"}
	case errors.Is(err, reportes.ErrImagenFormato):
		return errorHTTP{fiber.StatusUnsupportedMediaType, "IMAGEN_FORMATO",
			"la imagen debe ser JPEG"}
	case errors.Is(err, reportes.ErrImagenDimensiones):
		return errorHTTP{fiber.StatusUnprocessableEntity, "IMAGEN_DIMENSIONES",
			"dimensiones no válidas"}
	case errors.Is(err, reportes.ErrTextoRequerido):
		return errorHTTP{fiber.StatusBadRequest, "TEXTO_REQUERIDO",
			"el texto es obligatorio"}
	case errors.Is(err, reportes.ErrEnvioFallido):
		return errorHTTP{fiber.StatusBadGateway, "EVOLUTION_UNAVAILABLE",
			"no se pudo enviar el reporte"}
	default:
		return errorHTTP{fiber.StatusInternalServerError, "INTERNAL_ERROR",
			"error interno al procesar el reporte"}
	}
}

func responderError(c *fiber.Ctx, err error, legacy bool) error {
	mapped := mapearError(err)
	if legacy {
		return falloLegacy(c, mapped.status, mapped.message)
	}
	return fallo(c, mapped.status, mapped.code, mapped.message)
}

func fallo(c *fiber.Ctx, status int, code, message string) error {
	return httpx.Fail(c, status, code, message)
}

func falloLegacy(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{"success": false, "error": message})
}
