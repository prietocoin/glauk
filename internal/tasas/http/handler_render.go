package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/tasas/render"
	"glauk/internal/tasas/store"
)

// Calcular responde GET /calcular/:socio con la misma cartelera normalizada que
// alimenta al PNG. El panel ve exactamente lo que se publicaría.
func (d *Deps) Calcular(c *fiber.Ctx) error {
	cartelera, err := d.Service.CarteleraDe(c.Context(), c.Params("socio"))
	if errors.Is(err, store.ErrNotFound) {
		return NoEncontrado(c, "Socio no encontrado")
	}
	if err != nil {
		return ErrorInterno(c, err)
	}
	return OKFlat(c, fiber.Map{"datos": render.Normalizar(cartelera)})
}

// Render responde GET /render/:socio y POST /render con el PNG dibujado en memoria.
// El socio llega por ruta o por cuerpo; los errores son texto plano para el navegador.
func (d *Deps) Render(c *fiber.Ctx) error {
	cartelera, err := d.Service.CarteleraDe(c.Context(), nombreDeRender(c))
	if errors.Is(err, store.ErrNotFound) {
		return SinSocio(c)
	}
	if err != nil {
		return SinImagen(c)
	}
	png, err := render.GenerarCartelera(cartelera)
	if err != nil {
		return SinImagen(c)
	}
	return c.Type("png").Send(png)
}

// nombreDeRender unifica las dos entradas del endpoint de imagen.
func nombreDeRender(c *fiber.Ctx) string {
	if nombre := c.Params("socio"); nombre != "" {
		return nombre
	}
	var cuerpo struct {
		Socio string `json:"socio"`
	}
	if err := c.BodyParser(&cuerpo); err != nil {
		return ""
	}
	return cuerpo.Socio
}
