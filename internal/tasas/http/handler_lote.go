package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/tasas/store"
)

// GuardarLote responde POST /guardar-lote. El service sella CREATED_AT en UTC y
// deriva T### del último lote; el handler solo valida la forma del cuerpo.
func (d *Deps) GuardarLote(c *fiber.Ctx) error {
	var cuerpo struct {
		Tasas map[string]any `json:"tasas"`
	}
	if err := ParseBody(c, &cuerpo); err != nil || len(cuerpo.Tasas) == 0 {
		return Fail(c, fiber.StatusBadRequest, "Estructura de tasas inválida.")
	}
	lote, err := d.Service.GuardarLote(c.Context(), cuerpo.Tasas)
	if err != nil {
		return ErrorDeDominio(c, err)
	}
	return OKFlat(c, fiber.Map{"id_tasa": lote.IDTasa, "registro": lote})
}

// EliminarLote responde DELETE /lote/:id_tasa y distingue "borrado" de "no existía".
func (d *Deps) EliminarLote(c *fiber.Ctx) error {
	id := c.Params("id_tasa")
	borrado, err := d.Service.EliminarLote(c.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		return NoEncontrado(c, "No se encontró el lote "+id+".")
	}
	if err != nil {
		return ErrorDeDominio(c, err)
	}
	if !borrado {
		return NoEncontrado(c, "No se encontró el lote "+id+".")
	}
	return OKFlat(c, fiber.Map{"message": "Lote " + id + " eliminado correctamente."})
}
