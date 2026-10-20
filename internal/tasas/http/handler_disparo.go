package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/tasas/store"
)

// Disparar responde POST /disparar con la cartelera de un solo socio. El modo de
// prueba del cuerpo manda; si falta, el servicio usa el guardado en el perfil.
func (d *Deps) Disparar(c *fiber.Ctx) error {
	var cuerpo struct {
		Socio      string `json:"socio"`
		ModoPrueba *bool  `json:"modo_prueba"`
	}
	if err := ParseBody(c, &cuerpo); err != nil {
		return Fail(c, fiber.StatusBadRequest, "cuerpo de disparo inválido")
	}
	disparo, err := d.Service.Disparar(c.Context(), cuerpo.Socio, cuerpo.ModoPrueba)
	if errors.Is(err, store.ErrNotFound) {
		return NoEncontrado(c, "Socio no encontrado")
	}
	if err != nil {
		return ErrorInterno(c, err)
	}
	return OKFlat(c, fiber.Map{
		"socio":             disparo.Socio,
		"evolutionResponse": disparo.Respuesta,
	})
}
