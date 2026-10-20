package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"glauk/internal/comprobantes/domain"
	"glauk/internal/platform/postgres"
)

const queueListSQL = `
SELECT hash_largo, hash_corto, timestamp, nombre_socio_1, nombre_socio_2,
       conteo, estado, url_imagen
FROM cola_fb
ORDER BY timestamp DESC
LIMIT 100`

const queueUpdateSQL = `
UPDATE cola_fb
SET nombre_socio_1 = $1,
    nombre_socio_2 = $2,
    estado = COALESCE($3, estado),
    conteo = COALESCE($4, conteo),
    timestamp = COALESCE($5, timestamp)
WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($6))`

const deleteQueueComprobantesSQL = `
DELETE FROM comprobantes_fb WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($1))`

const deleteQueueAuditadosSQL = `
DELETE FROM comprobantes_auditados_fb WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($1))`

const deleteQueueSQL = `
DELETE FROM cola_fb WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($1))`

// Queue devuelve las últimas cien filas raw de la consola administrativa.
func (r *Repository) Queue(ctx context.Context) ([]domain.QueueItem, error) {
	rows, err := r.db.Query(ctx, queueListSQL)
	if err != nil {
		return nil, fmt.Errorf("consultar cola administrativa: %w", err)
	}
	defer rows.Close()
	items := make([]domain.QueueItem, 0)
	for rows.Next() {
		var item domain.QueueItem
		if err := rows.Scan(&item.HashLargo, &item.HashCorto, &item.Timestamp,
			&item.NombreSocio1, &item.NombreSocio2, &item.Conteo,
			&item.Estado, &item.URLImagen); err != nil {
			return nil, fmt.Errorf("leer cola administrativa: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("leer cola administrativa: %w", err)
	}
	return items, nil
}

// UpdateQueue conserva los campos omitidos mediante COALESCE.
func (r *Repository) UpdateQueue(ctx context.Context, hash string, data domain.QueueUpdate) (bool, error) {
	tag, err := r.db.Exec(ctx, queueUpdateSQL,
		data.NombreSocio1, data.NombreSocio2, data.Estado,
		data.Conteo, data.Timestamp, hash,
	)
	if err != nil {
		return false, fmt.Errorf("actualizar cola administrativa: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// DeleteQueue elimina las tres tablas raw en una única transacción.
func (r *Repository) DeleteQueue(ctx context.Context, hash string) error {
	return postgres.WithTx(ctx, r.db, pgx.TxOptions{}, func(tx pgx.Tx) error {
		for _, query := range []string{
			deleteQueueComprobantesSQL, deleteQueueAuditadosSQL, deleteQueueSQL,
		} {
			if _, err := tx.Exec(ctx, query, hash); err != nil {
				return fmt.Errorf("eliminar cola administrativa: %w", err)
			}
		}
		return nil
	})
}
