package http

import "github.com/gofiber/fiber/v2"

// BorradorHoo responde GET /hoo-borrador consultando el origen externo de tasas.
//
// No toca la base ni publica: el borrador es la propuesta que el operador revisa
// antes de guardar un lote. La dependencia puede ser nula cuando el origen no está
// configurado; en ese caso se devuelve un error explícito en vez de un pánico.
//
// Contrato:
//
//	200 {"success":true,"borrador":{"USD":1,"ARS":1250.5,"CREATED_AT":"..."}}
//	500 {"success":false,"error":"origen HOO no configurado"}
//
// El mapeo de los veinte campos vive en hoo/mapeo.go; esta capa solo transporta el
// resultado. Un fallo del origen se publica con su texto para que el operador sepa
// si fue red, estado HTTP o formato inesperado. La consulta respeta el contexto de
// la petición: si el panel cierra la conexión, se abandona la llamada externa.
// La respuesta no se cachea: el borrador representa el día en curso.
func (d *Deps) BorradorHoo(c *fiber.Ctx) error {
	if d.Hoo == nil {
		return Fail(c, fiber.StatusInternalServerError, "origen HOO no configurado")
	}
	cambios, err := d.Hoo.Obtener(c.Context())
	if err != nil {
		return ErrorInterno(c, err)
	}
	return OKFlat(c, fiber.Map{"borrador": cambios})
}
