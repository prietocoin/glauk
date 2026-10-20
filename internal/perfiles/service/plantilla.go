package service

import "context"

// Las tres operaciones corresponden a los botones del panel y son idempotentes:
// repetir apagar, memorizar o restaurar no corrompe recordar_mostrar. El
// servicio solo delega porque las reglas de conjunto viven en una única sentencia
// SQL; así no hay una ventana entre leer nombres y escribir estados.
//
// Contrato de las tres rutas:
//
//	POST /api/v1/perfiles/apagar-todos
//	POST /api/v1/perfiles/plantilla/memorizar
//	POST /api/v1/perfiles/plantilla/restaurar
//
// Ninguna acepta body. Devuelven {success:true,message} y nunca un perfil, de
// modo que el panel no confunde una operación masiva con una edición puntual.
// Si el repositorio falla, el error se propaga sin confirmar el lote.

// ApagarTodos desactiva mostrar.tasas en todos los socios salvo GENERAL. La
// operación es masiva y atómica en el repositorio; el servicio no reescribe la
// regla de protección en Go para no tener dos definiciones del mismo lote ni
// una ventana de inconsistencia entre la lectura y la escritura masiva.
func (s *Service) ApagarTodos(ctx context.Context) error {
	return s.escritor.ApagarTodos(ctx)
}

// Memorizar guarda una copia del estado actual de mostrar en recordar_mostrar.
// No activa ni desactiva socios: solo congela la plantilla vigente.
func (s *Service) Memorizar(ctx context.Context) error {
	return s.escritor.Memorizar(ctx)
}

// Restaurar recupera la última plantilla y vuelve a forzar GENERAL activo. La
// columna recordar_mostrar se crea por migración, nunca en tiempo de request.
// Si nunca se memorizó, el SQL usa el default apagado para los socios y mantiene
// GENERAL encendido, de modo que la operación no deja el panel inutilizable.
func (s *Service) Restaurar(ctx context.Context) error {
	return s.escritor.Restaurar(ctx)
}
