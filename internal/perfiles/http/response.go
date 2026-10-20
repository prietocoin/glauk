package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"

	"glauk/internal/perfiles/service"
)

// ErrServicioNoConfigurado se devuelve al montar el router sin dependencias.
var ErrServicioNoConfigurado = errors.New("perfiles: servicio no configurado")

// exito escribe el envelope de escritura que espera el panel.
func exito(c *fiber.Ctx, data any) error {
	return c.JSON(fiber.Map{"success": true, "data": data})
}

// mensaje confirma operaciones masivas sin inventar un payload de perfil.
func mensaje(c *fiber.Ctx, texto string) error {
	return c.JSON(fiber.Map{"success": true, "message": texto})
}

// fallo conserva el envelope histórico de Perfiles y añade el código estable.
func fallo(c *fiber.Ctx, status int, code, texto string) error {
	return c.Status(status).JSON(fiber.Map{
		"success": false,
		"error":   texto,
		"code":    code,
	})
}

// responderError traduce errores de dominio y de persistencia a HTTP.
func responderError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrNombreRequerido):
		return fallo(c, fiber.StatusBadRequest, "NOMBRE_REQUERIDO", err.Error())
	case errors.Is(err, service.ErrEntidadProtegida):
		return fallo(c, fiber.StatusBadRequest, "ENTIDAD_PROTEGIDA", err.Error())
	case errors.Is(err, pgx.ErrNoRows):
		return fallo(c, fiber.StatusNotFound, "NO_ENCONTRADO", "el socio no existe")
	default:
		return fallo(c, fiber.StatusInternalServerError, "ERROR_INTERNO", "no se pudo completar la operación")
	}
}
