package http

import (
	"github.com/gofiber/fiber/v2"

	"glauk/internal/perfiles/model"
)

// Guardar responde POST /api/v1/perfiles/guardar.
func (d Deps) Guardar(c *fiber.Ctx) error {
	var req model.GuardarRequest
	if err := c.BodyParser(&req); err != nil {
		return fallo(c, fiber.StatusBadRequest, "JSON_INVALIDO", "payload JSON inválido")
	}
	perfil, err := d.Perfiles.Guardar(c.UserContext(), req)
	if err != nil {
		return responderError(c, err)
	}
	return exito(c, perfil)
}

// Toggle responde PATCH /api/v1/perfiles/:nombre/estado.
func (d Deps) Toggle(c *fiber.Ctx) error {
	var body struct {
		Activo bool `json:"activo"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fallo(c, fiber.StatusBadRequest, "JSON_INVALIDO", "payload JSON inválido")
	}
	perfil, err := d.Perfiles.Toggle(c.UserContext(), c.Params("nombre"), body.Activo)
	if err != nil {
		return responderError(c, err)
	}
	return exito(c, perfil)
}

// Eliminar responde DELETE /api/v1/perfiles/:nombre.
func (d Deps) Eliminar(c *fiber.Ctx) error {
	if err := d.Perfiles.Eliminar(c.UserContext(), c.Params("nombre")); err != nil {
		return responderError(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

// ApagarTodos responde POST /api/v1/perfiles/apagar-todos.
func (d Deps) ApagarTodos(c *fiber.Ctx) error {
	if err := d.Perfiles.ApagarTodos(c.UserContext()); err != nil {
		return responderError(c, err)
	}
	return mensaje(c, "socios apagados; GENERAL permanece activo")
}
