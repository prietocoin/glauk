package http

import "github.com/gofiber/fiber/v2"

// Listar responde GET /api/v1/perfiles con un array raíz, contrato del panel.
//
// No envuelve la respuesta en {success,data}: el frontend de Perfiles recorre el
// array directamente y depende de esa forma desde la versión Express. El campo
// activo de cada fila llega derivado por H-02, nunca leído de una columna.
func (d Deps) Listar(c *fiber.Ctx) error {
	perfiles, err := d.Perfiles.Listar(c.UserContext())
	if err != nil {
		return responderError(c, err)
	}
	return c.JSON(perfiles)
}

// Activos responde GET /api/v1/perfiles/activos con los nombres publicables.
// La consulta filtra por mostrar.tasas en SQL; el handler no reinterpreta la
// regla H-02 ni consulta la columna legada. Si la lista está vacía, el cuerpo es
// [] y el cliente puede recorrerlo sin una comprobación extra.
//
// También devuelve un array raíz de strings para conservar el contrato legado
// que consumen Tasas y Comprobantes. Un directorio vacío se serializa como []
// y no como null, porque ambos clientes iteran la respuesta sin comprobar nulo.
func (d Deps) Activos(c *fiber.Ctx) error {
	nombres, err := d.Perfiles.Activos(c.UserContext())
	if err != nil {
		return responderError(c, err)
	}
	return c.JSON(nombres)
}
