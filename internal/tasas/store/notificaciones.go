package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// sqlInsertNotificacion registra una notificación del lote indicado. La columna de
// fecha la pone la base con NOW(), igual que el servicio original, para que el
// momento registrado sea el de la base y no el del proceso que la solicitó.
const sqlInsertNotificacion = `INSERT INTO notificaciones_tasas (id_tasa, creado_en) VALUES ($1, NOW()) RETURNING id_tasa`

// RecordNotification registra una notificación de tasas para el lote indicado.
//
// Solo se envía el identificador del lote: las demás columnas las genera la base, y
// el flujo de remesas cuelga de sus disparadores, no de esta llamada. Por eso se
// devuelve únicamente el identificador y no la fila entera: es el único dato que el
// store puede afirmar sin conocer el resto del esquema, y basta para que quien
// responde confirme qué lote se notificó.
func RecordNotification(ctx context.Context, q Querier, idTasa string) (string, error) {
	return queryOne(ctx, q, sqlInsertNotificacion, []any{idTasa}, scanNotification)
}

// scanNotification lee la fila que devuelve la inserción, que es un solo campo.
func scanNotification(rows pgx.Rows) (string, error) {
	var id string
	if err := rows.Scan(&id); err != nil {
		return "", fmt.Errorf("escaneando la notificación: %w", err)
	}
	return id, nil
}
