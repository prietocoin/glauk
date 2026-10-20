package http

import "github.com/gofiber/fiber/v2"

// NotificarRemesas responde POST /notificar-remesas. Registra el lote vigente y
// devuelve solo su identificador: el disparador de remesas cuelga de la tabla y no
// necesita que esta capa conozca el resto del esquema.
//
// Contrato:
//
//	200 {"success":true,"notificacion":{"id_tasa":"T053"}}
//	500 {"success":false,"error":"..."}
//
// El cuerpo de entrada se ignora a propósito: el lote notificado siempre es el
// vigente, igual que en el servicio Node. No se publica ninguna cartelera aquí;
// solo se deja la fila que dispara la automatización externa.
//
// Si la tabla todavía no tiene lotes, se registra T000. Es el mismo valor que el
// monitor publica en ese estado y evita que la automatización reciba una cadena
// vacía como identificador. El cuerpo de la respuesta conserva la forma anidada
// que ya consumía n8n.
func (d *Deps) NotificarRemesas(c *fiber.Ctx) error {
	notificacion, err := d.Service.NotificarRemesas(c.Context())
	if err != nil {
		return ErrorInterno(c, err)
	}
	return OKFlat(c, fiber.Map{
		"notificacion": fiber.Map{"id_tasa": notificacion.IDTasa},
	})
}
