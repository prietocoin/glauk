package store

import (
	"context"
	"fmt"

	"glauk/internal/tasas/domain"
)

// Sentencias de escritura de tasas_glaukov.
//
// El lote se inserta con el identificador ya calculado por NextLoteID y no se
// actualiza nunca: un lote publicado es historia, y corregirlo cambiaría lo que
// recibieron los socios.
const (
	sqlInsertLote = `INSERT INTO tasas_glaukov (id_tasa, tasas) VALUES ($1, $2) RETURNING id_tasa, tasas`
	sqlDeleteLote = `DELETE FROM tasas_glaukov WHERE LOWER(id_tasa) = LOWER($1)`
)

// InsertLote guarda un lote nuevo y devuelve la fila tal como quedó en la tabla.
//
// Devolver la fila guardada, y no el valor recibido, es lo que permite confirmar
// que la base aceptó exactamente lo que se envió.
func InsertLote(ctx context.Context, q Querier, id string, tasas map[string]any) (domain.Lote, error) {
	raw, err := encodeJSON(tasas)
	if err != nil {
		return domain.Lote{}, err
	}
	return queryOne(ctx, q, sqlInsertLote, []any{id, raw}, scanLote)
}

// DeleteLote elimina un lote por su identificador e informa si existía.
//
// El booleano existe para que la capa HTTP distinga "borrado" de "no encontré nada
// que borrar": el original hacía esa misma distinción con el conteo de filas, pero
// con un RETURNING que descartaba.
func DeleteLote(ctx context.Context, q Querier, id string) (bool, error) {
	tag, err := q.Exec(ctx, sqlDeleteLote, id)
	if err != nil {
		return false, fmt.Errorf("eliminando el lote %s: %w", id, err)
	}
	return tag.RowsAffected() > 0, nil
}
