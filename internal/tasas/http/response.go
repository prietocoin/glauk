package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/tasas/service"
	"glauk/internal/tasas/store"
)

// OKFlat responde 200 con success:true al mismo nivel que los campos del endpoint.
// El panel lee siempre esa marca, venga la respuesta de la base o de un cálculo.
func OKFlat(c *fiber.Ctx, campos fiber.Map) error {
	campos["success"] = true
	return c.JSON(campos)
}

// Fail responde el código indicado con la forma estable {success:false,error}.
func Fail(c *fiber.Ctx, status int, mensaje string) error {
	return c.Status(status).JSON(fiber.Map{"success": false, "error": mensaje})
}

// NoEncontrado conserva el texto exacto que publicaba cada endpoint.
func NoEncontrado(c *fiber.Ctx, mensaje string) error {
	return Fail(c, fiber.StatusNotFound, mensaje)
}

// ErrorInterno expone la causa como el servicio Node, sin inventar un mensaje.
func ErrorInterno(c *fiber.Ctx, err error) error {
	return Fail(c, fiber.StatusInternalServerError, err.Error())
}

// ErrorDeDominio traduce los centinelas del servicio y del store a su código HTTP.
func ErrorDeDominio(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return NoEncontrado(c, "Socio no encontrado")
	case errors.Is(err, service.ErrSinLotes),
		errors.Is(err, service.ErrLoteInvalido),
		errors.Is(err, service.ErrTasasInvalidas):
		return Fail(c, fiber.StatusBadRequest, err.Error())
	default:
		return ErrorInterno(c, err)
	}
}

// ParseBody interpreta JSON cuando hay cuerpo. Un cuerpo vacío es válido: varios
// endpoints del panel se llaman sin datos, como hacía Express.
func ParseBody(c *fiber.Ctx, destino any) error {
	if len(c.Body()) == 0 {
		return nil
	}
	return c.BodyParser(destino)
}

// SinSocio y SinImagen conservan las respuestas de texto plano del render: su
// cliente es el navegador que pinta la imagen, no el lector JSON del panel.
func SinSocio(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).SendString("Socio no encontrado")
}

func SinImagen(c *fiber.Ctx) error {
	return c.Status(fiber.StatusInternalServerError).SendString("Error generando PNG")
}
