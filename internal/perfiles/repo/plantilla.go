package repo

import "context"

// Operaciones masivas de plantilla. Cada sentencia es atómica; recordar_mostrar
// se crea por migración y nunca en tiempo de request. El repositorio no decide
// qué socios son matrices: las reglas están fijadas en el SQL para que el lote
// sea una sola sentencia y no una carrera entre lecturas y escrituras de Go.

// ApagarTodos desactiva tasas en todos los socios salvo GENERAL. La entidad
// matriz permanece visible porque el panel la necesita para editar herencia.
func (w *Writer) ApagarTodos(ctx context.Context) error {
	_, err := w.db.Exec(ctx, ApagarTodosSQL)
	return err
}

// Memorizar copia mostrar a recordar_mostrar para una restauración posterior.
// La columna es parte del esquema por 0001_init.sql; no se crea aquí ni se
// rellena con un DEFAULT, porque una plantilla vacía debe ser un estado válido.
func (w *Writer) Memorizar(ctx context.Context) error {
	_, err := w.db.Exec(ctx, MemorizarSQL)
	return err
}

// Restaurar recupera recordar_mostrar y mantiene GENERAL activo. Si una fila no
// tiene copia, el COALESCE del SQL aplica el default apagado sin romper el lote.
func (w *Writer) Restaurar(ctx context.Context) error {
	_, err := w.db.Exec(ctx, RestaurarSQL)
	return err
}
