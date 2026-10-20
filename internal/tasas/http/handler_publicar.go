package http

import "github.com/gofiber/fiber/v2"

// PublicarLote responde POST /publicar-lote y POST /disparar-lote.
//
// Ambos endpoints comparten handler porque el servicio Node también los compartía:
// publicar el lote vigente y disparar el lote general son la misma operación. La
// respuesta incluye el detalle por socio; los errores parciales no son un 500,
// porque el resto de la publicación sí se completó.
//
// Sin lotes el servicio devuelve ErrSinLotes y la respuesta es 400: es una entrada
// incompleta, no un fallo del servidor.
func (d *Deps) PublicarLote(c *fiber.Ctx) error {
	var cuerpo struct {
		ModoPruebaGlobal bool `json:"modo_prueba_global"`
	}
	if err := ParseBody(c, &cuerpo); err != nil {
		return Fail(c, fiber.StatusBadRequest, "cuerpo de publicación inválido")
	}
	publicacion, err := d.Service.Publicar(c.Context(), cuerpo.ModoPruebaGlobal)
	if err != nil {
		return ErrorDeDominio(c, err)
	}
	return OKFlat(c, fiber.Map{
		"id_tasa":    publicacion.IDTasa,
		"procesados": publicacion.Procesados,
		"detalle":    publicacion.Detalle,
	})
}
