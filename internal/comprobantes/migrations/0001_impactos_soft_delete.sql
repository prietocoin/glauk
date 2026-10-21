-- Fase 8: borrado lógico de impactos_raw.
-- La eliminación de un comprobante conserva la evidencia cruda de WhatsApp,
-- pero la aparta del listado y de los filtros de Reportes.
-- Idempotente: se puede aplicar varias veces sin alterar datos existentes.
ALTER TABLE impactos_raw ADD COLUMN IF NOT EXISTS eliminado_en TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS impactos_raw_eliminado_en_idx ON impactos_raw (eliminado_en);

CREATE INDEX IF NOT EXISTS impactos_raw_vigentes_idx
    ON impactos_raw (LOWER(BTRIM(hash_largo)))
    WHERE eliminado_en IS NULL;
