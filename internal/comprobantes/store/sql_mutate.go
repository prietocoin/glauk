package store

const upsertLiquidacionSQL = `
INSERT INTO comprobantes_liq (
    hash_largo, socio_1, tipo_op1, monto_1, tasa_1, me1,
    socio_2, tipo_op2, monto_2, tasa_2, me2, lote_tasa, actualizado_en
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
ON CONFLICT (hash_largo) DO UPDATE SET
    socio_1 = EXCLUDED.socio_1, tipo_op1 = EXCLUDED.tipo_op1,
    monto_1 = EXCLUDED.monto_1, tasa_1 = EXCLUDED.tasa_1, me1 = EXCLUDED.me1,
    socio_2 = EXCLUDED.socio_2, tipo_op2 = EXCLUDED.tipo_op2,
    monto_2 = EXCLUDED.monto_2, tasa_2 = EXCLUDED.tasa_2, me2 = EXCLUDED.me2,
    lote_tasa = EXCLUDED.lote_tasa, actualizado_en = NOW()`

const updateRawSQL = `
UPDATE comprobantes_raw
SET monto = $1, moneda = NULLIF($2, ''), banco = NULLIF($3, ''),
    referencia = NULLIF($4, ''), titular = NULLIF($5, '')
WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($6))`

const deleteLiquidacionSQL = `
DELETE FROM comprobantes_liq WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($1))`

// MarkImpactosSQL aplica borrado lógico para no destruir el historial de origen.
const markImpactosSQL = `
UPDATE impactos_raw SET eliminado_en = NOW()
WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($1))`

const deleteComprobanteSQL = `
DELETE FROM comprobantes_raw WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($1))`

const markReprocessingSQL = `
UPDATE comprobantes_raw
SET estado_ia = 'RE-PROCESANDO', procesado_ia = FALSE
WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($1))`
