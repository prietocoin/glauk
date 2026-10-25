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

// sweepOrphanImpactosSQL aparta del pipeline la evidencia de WhatsApp que no
// logró consolidar un binomio 2x en la ventana de gracia. Se marca
// eliminado_en en vez de borrar: la evidencia sigue auditable, pero deja de
// competir con las lecturas del listado y de Reportes.
//
// La antigüedad se mide con comprobantes_raw.creado_en, que es el sello
// estable del recibo. Un comprobante_liq con socio_2 NULL, vacío o GENERAL no
// cuenta como binomio completo.
const sweepOrphanImpactosSQL = `
UPDATE impactos_raw AS i
SET eliminado_en = NOW()
FROM comprobantes_raw AS c
WHERE i.eliminado_en IS NULL
  AND i.hash_largo IS NOT NULL
  AND btrim(i.hash_largo) <> ''
  AND lower(btrim(i.hash_largo)) = lower(btrim(c.hash_largo))
  AND c.creado_en < NOW() - ($1::bigint * INTERVAL '1 second')
  AND NOT EXISTS (
    SELECT 1
    FROM comprobantes_liq AS l
    WHERE lower(btrim(l.hash_largo)) = lower(btrim(c.hash_largo))
      AND l.socio_2 IS NOT NULL
      AND btrim(l.socio_2) <> ''
      AND UPPER(btrim(l.socio_2)) <> 'GENERAL'
  )`

// purgeOrphanComprobantesSQL elimina los recibos crudos que, después de la
// ventana de gracia, no tienen binomio completo y ya no conservan impactos
// vigentes. No toca comprobantes_liq: el snapshot canónico nunca se borra por
// un barrido de higiene.
const purgeOrphanComprobantesSQL = `
DELETE FROM comprobantes_raw AS c
WHERE c.creado_en < NOW() - ($1::bigint * INTERVAL '1 second')
  AND NOT EXISTS (
    SELECT 1
    FROM comprobantes_liq AS l
    WHERE lower(btrim(l.hash_largo)) = lower(btrim(c.hash_largo))
      AND l.socio_2 IS NOT NULL
      AND btrim(l.socio_2) <> ''
      AND UPPER(btrim(l.socio_2)) <> 'GENERAL'
  )
  AND NOT EXISTS (
    SELECT 1
    FROM impactos_raw AS i
    WHERE lower(btrim(i.hash_largo)) = lower(btrim(c.hash_largo))
      AND i.eliminado_en IS NULL
  )`

// sweepNonPositiveImpactosSQL retira la evidencia de WhatsApp de los recibos
// con monto no positivo antes de borrar sus filas canónicas. Se usa borrado
// lógico para conservar la trazabilidad del origen.
const sweepNonPositiveImpactosSQL = `
UPDATE impactos_raw AS i
SET eliminado_en = NOW()
FROM comprobantes_raw AS c
WHERE i.eliminado_en IS NULL
  AND i.hash_largo IS NOT NULL
  AND btrim(i.hash_largo) <> ''
  AND lower(btrim(i.hash_largo)) = lower(btrim(c.hash_largo))
  AND c.monto <= 0
  AND c.creado_en < NOW() - ($1::bigint * INTERVAL '1 second')`

// purgeNonPositiveLiquidacionesSQL borra el snapshot de los recibos con monto
// no positivo. comprobantes_liq no tiene creado_en: la antigüedad se toma del
// comprobante_raw asociado, que es el sello estable del recibo. Debe ejecutarse
// antes del DELETE de comprobantes_raw para conservar el join.
const purgeNonPositiveLiquidacionesSQL = `
DELETE FROM comprobantes_liq AS l
USING comprobantes_raw AS c
WHERE lower(btrim(l.hash_largo)) = lower(btrim(c.hash_largo))
  AND c.monto <= 0
  AND c.creado_en < NOW() - ($1::bigint * INTERVAL '1 second')`

// purgeNonPositiveComprobantesSQL elimina los recibos crudos con monto no
// positivo después de la ventana de gracia.
const purgeNonPositiveComprobantesSQL = `
DELETE FROM comprobantes_raw
WHERE monto <= 0
  AND creado_en < NOW() - ($1::bigint * INTERVAL '1 second')`

// OrphanSweep resume las filas tocadas por SweepOrphans.
type OrphanSweep struct {
	Impactos     int
	Comprobantes int
}

// NonPositiveSweep resume las filas tocadas por SweepNonPositive.
type NonPositiveSweep struct {
	Impactos      int
	Liquidaciones int
	Comprobantes  int
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

// SweepOrphans aplica la retención P-09: los recibos crudos que no lograron
// consolidar un binomio 2x dentro de la ventana se purgan, y su evidencia de
// WhatsApp se marca como eliminada. Es idempotente y seguro de repetir en cada
// tick: ambas sentencias vuelven a evaluar el estado actual.
func (r *Repository) SweepOrphans(ctx context.Context, olderThan time.Duration) (OrphanSweep, error) {
	if olderThan <= 0 {
		return OrphanSweep{}, fmt.Errorf("ingesta: la antigüedad de huérfanos debe ser positiva")
	}
	seconds := int64(olderThan / time.Second)
	if seconds < 1 {
		seconds = 1
	}

	tagImpactos, err := r.db.Exec(ctx, sweepOrphanImpactosSQL, seconds)
	if err != nil {
		return OrphanSweep{}, r.sqlFail("sweep_impactos_huerfanos",
			fmt.Errorf("ingesta: marcar impactos huérfanos: %w", err))
	}
	tagComprobantes, err := r.db.Exec(ctx, purgeOrphanComprobantesSQL, seconds)
	if err != nil {
		return OrphanSweep{Impactos: int(tagImpactos.RowsAffected())},
			r.sqlFail("purge_comprobantes_huerfanos",
				fmt.Errorf("ingesta: purgar comprobantes huérfanos: %w", err))
	}
	return OrphanSweep{
		Impactos:     int(tagImpactos.RowsAffected()),
		Comprobantes: int(tagComprobantes.RowsAffected()),
	}, nil
}

// SweepNonPositive aplica la retención de montos no positivos: marca la
// evidencia, borra el snapshot liquidado y luego el recibo crudo. La antigüedad
// se mide con comprobantes_raw.creado_en porque comprobantes_liq no tiene un
// sello temporal propio en el contrato mínimo. Es idempotente: si una sentencia
// falla, el siguiente tick retoma desde el estado actual.
func (r *Repository) SweepNonPositive(ctx context.Context, olderThan time.Duration) (NonPositiveSweep, error) {
	if olderThan <= 0 {
		return NonPositiveSweep{}, fmt.Errorf("ingesta: la antigüedad de montos no positivos debe ser positiva")
	}
	seconds := int64(olderThan / time.Second)
	if seconds < 1 {
		seconds = 1
	}

	tagImpactos, err := r.db.Exec(ctx, sweepNonPositiveImpactosSQL, seconds)
	if err != nil {
		return NonPositiveSweep{}, r.sqlFail("sweep_impactos_monto_no_positivo",
			fmt.Errorf("ingesta: marcar impactos con monto no positivo: %w", err))
	}
	resultado := NonPositiveSweep{Impactos: int(tagImpactos.RowsAffected())}

	tagLiquidaciones, err := r.db.Exec(ctx, purgeNonPositiveLiquidacionesSQL, seconds)
	if err != nil {
		return resultado, r.sqlFail("purge_liquidaciones_monto_no_positivo",
			fmt.Errorf("ingesta: purgar liquidaciones con monto no positivo: %w", err))
	}
	resultado.Liquidaciones = int(tagLiquidaciones.RowsAffected())

	tagComprobantes, err := r.db.Exec(ctx, purgeNonPositiveComprobantesSQL, seconds)
	if err != nil {
		return resultado, r.sqlFail("purge_comprobantes_monto_no_positivo",
			fmt.Errorf("ingesta: purgar comprobantes con monto no positivo: %w", err))
	}
	resultado.Comprobantes = int(tagComprobantes.RowsAffected())
	return resultado, nil
}
