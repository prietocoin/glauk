package ingesta

import (
	"context"
	"fmt"
	"time"
)

// normalizeRegistrosHashSQL sanea las claves históricas del pipeline. El
// remitHub anterior escribía hashes en mayúsculas; el Core escribe minúsculas.
// Solo actualiza filas que no colisionan con una versión minúscula ya
// existente: si el duplicado existe, se conserva como evidencia y el barrido
// posterior lo deja en FALLO.
const normalizeRegistrosHashSQL = `
UPDATE registros_raw AS r
SET hash_largo = lower(btrim(r.hash_largo))
WHERE r.hash_largo IS NOT NULL
  AND btrim(r.hash_largo) <> ''
  AND r.hash_largo <> lower(btrim(r.hash_largo))
  AND NOT EXISTS (
    SELECT 1
    FROM registros_raw AS x
    WHERE x.hash_largo = lower(btrim(r.hash_largo))
  )`

// sweepStaleJobsSQL libera los trabajos que quedaron encolados por un proceso
// que ya no existe. registros_raw no tiene una columna de actualización en el
// contrato actual: timestamp_msg es el sello del mensaje y, para una fila
// EN_COLA, el instante en que el binomio quedó completo. Un timestamp 0 se
// considera huérfano. FALLO es el estado que Claim acepta para reintentar.
const sweepStaleJobsSQL = `
UPDATE registros_raw
SET estado = 'FALLO'
WHERE estado = 'EN_COLA'
  AND (
    COALESCE(timestamp_msg, 0) = 0
    OR to_timestamp(
         CASE
           WHEN timestamp_msg > 100000000000
             THEN (timestamp_msg / 1000.0)::double precision
           ELSE timestamp_msg::double precision
         END
       ) < NOW() - ($1::bigint * INTERVAL '1 second')
  )`

// NormalizeRegistrosHash convierte las claves heredadas a minúsculas antes de
// que el dispatcher empiece a trabajar. Es idempotente y se ejecuta una sola
// vez por arranque.
func (r *Repository) NormalizeRegistrosHash(ctx context.Context) (int, error) {
	tag, err := r.db.Exec(ctx, normalizeRegistrosHashSQL)
	if err != nil {
		return 0, r.sqlFail("normalize_registros_raw_hash",
			fmt.Errorf("ingesta: normalizar hashes de registros_raw: %w", err))
	}
	return int(tag.RowsAffected()), nil
}

// SweepStaleJobs transiciona de EN_COLA a FALLO los trabajos que superaron la
// ventana de gracia. No borra filas: la fila queda auditable y Claim puede
// volver a encolarla si el webhook se reenvía.
func (r *Repository) SweepStaleJobs(ctx context.Context, olderThan time.Duration) (int, error) {
	if olderThan <= 0 {
		return 0, fmt.Errorf("ingesta: la antigüedad de barrido debe ser positiva")
	}
	seconds := int64(olderThan / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	tag, err := r.db.Exec(ctx, sweepStaleJobsSQL, seconds)
	if err != nil {
		return 0, r.sqlFail("sweep_registros_raw",
			fmt.Errorf("ingesta: barrer trabajos EN_COLA: %w", err))
	}
	return int(tag.RowsAffected()), nil
}
