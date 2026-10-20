package http

import "github.com/gofiber/fiber/v2"

// Register monta exactamente las ocho rutas canónicas de §5.4.1:
//
//	GET    /api/v1/perfiles
//	GET    /api/v1/perfiles/activos
//	POST   /api/v1/perfiles/guardar
//	POST   /api/v1/perfiles/apagar-todos
//	POST   /api/v1/perfiles/plantilla/memorizar
//	POST   /api/v1/perfiles/plantilla/restaurar
//	PATCH  /api/v1/perfiles/:nombre/estado
//	DELETE /api/v1/perfiles/:nombre
//
// No registra alias: esa responsabilidad queda en cmd/server, que reutiliza el
// mismo Deps para conservar el contrato de URL histórico sin duplicar reglas.
func Register(app *fiber.App, deps Deps) error {
	if err := deps.Valida(); err != nil {
		return err
	}
	app.Get(BasePath, deps.Listar)
	app.Get(BasePath+"/activos", deps.Activos)
	app.Post(BasePath+"/guardar", deps.Guardar)
	app.Post(BasePath+"/apagar-todos", deps.ApagarTodos)
	app.Post(BasePath+"/plantilla/memorizar", deps.Memorizar)
	app.Post(BasePath+"/plantilla/restaurar", deps.Restaurar)
	app.Patch(BasePath+"/:nombre/estado", deps.Toggle)
	app.Delete(BasePath+"/:nombre", deps.Eliminar)
	return nil
}
