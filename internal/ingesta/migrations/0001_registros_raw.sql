-- Pipeline nativo de ingesta (remitHub + glaukapi -> Core).
-- Idempotente: se puede aplicar dos veces sin borrar ni reescribir filas.
-- Requiere haber aplicado 0001_impactos_soft_delete.sql.

BEGIN;

CREATE TABLE IF NOT EXISTS registros_raw (
    hash_largo    TEXT PRIMARY KEY,
    hash_corto    TEXT,
    grupo_raw     TEXT,
    usuario_raw   TEXT,
    nombre_push   TEXT,
    caption       TEXT,
    timestamp_msg BIGINT,
    es_imagen     BOOLEAN NOT NULL DEFAULT TRUE,
    instancia     TEXT NOT NULL DEFAULT 'glauk',
    url_imagen    TEXT,
    conteo        INTEGER NOT NULL DEFAULT 0,
    estado        TEXT NOT NULL DEFAULT 'RECIBIDO'
);

-- La tabla puede venir del servicio Node con un subconjunto de columnas.
ALTER TABLE registros_raw ADD COLUMN IF NOT EXISTS hash_corto TEXT;
ALTER TABLE registros_raw ADD COLUMN IF NOT EXISTS grupo_raw TEXT;
ALTER TABLE registros_raw ADD COLUMN IF NOT EXISTS usuario_raw TEXT;
ALTER TABLE registros_raw ADD COLUMN IF NOT EXISTS nombre_push TEXT;
ALTER TABLE registros_raw ADD COLUMN IF NOT EXISTS caption TEXT;
ALTER TABLE registros_raw ADD COLUMN IF NOT EXISTS timestamp_msg BIGINT;
ALTER TABLE registros_raw ADD COLUMN IF NOT EXISTS es_imagen BOOLEAN DEFAULT TRUE;
ALTER TABLE registros_raw ADD COLUMN IF NOT EXISTS instancia TEXT DEFAULT 'glauk';
ALTER TABLE registros_raw ADD COLUMN IF NOT EXISTS url_imagen TEXT;
ALTER TABLE registros_raw ADD COLUMN IF NOT EXISTS conteo INTEGER DEFAULT 0;
ALTER TABLE registros_raw ADD COLUMN IF NOT EXISTS estado TEXT DEFAULT 'RECIBIDO';

ALTER TABLE impactos_raw ADD COLUMN IF NOT EXISTS eliminado_en TIMESTAMPTZ;
ALTER TABLE impactos_raw ADD COLUMN IF NOT EXISTS nombre_push TEXT;

ALTER TABLE comprobantes_raw ADD COLUMN IF NOT EXISTS estado_ia TEXT;
ALTER TABLE comprobantes_raw ADD COLUMN IF NOT EXISTS actualizado_en TIMESTAMPTZ;
ALTER TABLE comprobantes_liq ADD COLUMN IF NOT EXISTS actualizado_en TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS ix_impactos_raw_binomio
    ON impactos_raw (LOWER(TRIM(hash_largo)), eliminado_en);

CREATE INDEX IF NOT EXISTS ix_registros_raw_estado
    ON registros_raw (estado);

COMMIT;
