package http

import "github.com/gofiber/fiber/v2"

// Monitor responde GET /monitor con el lote vigente. Sin lotes devuelve T000 y un
// objeto vacío: el panel dibuja un estado vacío en lugar de mostrar un 404.
//
// El endpoint es de solo lectura y nunca calcula carteleras: publica el lote tal
// como está guardado, para que el operador vea la fuente de verdad antes de editar.
//
// Contrato de salida:
//
//	{"success":true,"id_tasa":"T052","tasas":{"USD":1,"ARS":1250.5}}
//	{"success":true,"id_tasa":"T000","tasas":{}}
//
// El identificador y las tasas vienen del servicio; el handler no consulta el
// store ni conoce el esquema. Un fallo del motor se publica como 500 con el texto
// original, porque el panel muestra ese mensaje al operador.
//
// No se aplica el filtro H-02 aquí: el monitor muestra el lote, no la lista de
// destinatarios. La regla de actividad se aplica al publicar, que es donde se
// decide qué socios reciben una cartelera. El orden de las tasas es el del JSONB
// guardado, que el panel recorre sin exigir una secuencia concreta.
func (d *Deps) Monitor(c *fiber.Ctx) error {
	lote, err := d.Service.Vigente(c.Context())
	if err != nil {
		return ErrorInterno(c, err)
	}
	return OKFlat(c, fiber.Map{"id_tasa": lote.IDTasa, "tasas": lote.Tasas})
}
