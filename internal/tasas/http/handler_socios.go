package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/tasas/domain"
	"glauk/internal/tasas/store"
)

// cambiosDeSocio es el cuerpo parcial de PUT /socios/:socio. Los punteros distinguen
// "no lo toques" de "ponlo en falso"; un campo ausente no se escribe.
type cambiosDeSocio struct {
	Monedas    map[string]*domain.Moneda `json:"monedas"`
	Activo     *bool                     `json:"activo"`
	ModoPrueba *bool                     `json:"modo_prueba"`
	Rol        *string                   `json:"rol"`
}

// Socios responde GET /socios con la lista normalizada: monedas nunca es null y el
// rol vacío se publica como SOCIO.
func (d *Deps) Socios(c *fiber.Ctx) error {
	socios, err := d.Service.Socios(c.Context())
	if err != nil {
		return ErrorInterno(c, err)
	}
	return OKFlat(c, fiber.Map{"socios": socios})
}

// GuardarSocio responde PUT /socios/:socio. Un socio inexistente devuelve 404; el
// legado respondía éxito con un objeto vacío y el panel creía haber guardado.
func (d *Deps) GuardarSocio(c *fiber.Ctx) error {
	var cambios cambiosDeSocio
	if err := ParseBody(c, &cambios); err != nil {
		return Fail(c, fiber.StatusBadRequest, "cuerpo de socio inválido")
	}
	socio, err := d.Service.GuardarSocio(c.Context(), c.Params("socio"), store.SocioUpdate{
		Monedas:    cambios.Monedas,
		Activo:     cambios.Activo,
		ModoPrueba: cambios.ModoPrueba,
		Rol:        cambios.Rol,
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return NoEncontrado(c, "Socio no encontrado")
		}
		return ErrorInterno(c, err)
	}
	return OKFlat(c, fiber.Map{"socio": socio})
}
