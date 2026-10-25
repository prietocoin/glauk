-- GET /api/v1/comprobantes: índices de soporte para el listado y el filtro socio.
--
-- La consulta une comprobantes_raw con comprobantes_liq y los impactos usando
-- lower(btrim(hash_largo)). Sin índices funcionales, PostgreSQL recorre la
-- tabla histórica completa para cada fila y la consulta supera el
-- statement_timeout de 10 s (HTTP 500). Idempotente.

CREATE INDEX IF NOT EXISTS comprobantes_raw_creado_en_idx
    ON comprobantes_raw (creado_en);

CREATE INDEX IF NOT EXISTS comprobantes_raw_instancia_creado_idx
    ON comprobantes_raw (lower(btrim(instancia)), creado_en);

CREATE INDEX IF NOT EXISTS comprobantes_raw_hash_lower_idx
    ON comprobantes_raw (lower(btrim(hash_largo)));

CREATE INDEX IF NOT EXISTS comprobantes_liq_hash_lower_idx
    ON comprobantes_liq (lower(btrim(hash_largo)));

CREATE INDEX IF NOT EXISTS impactos_raw_vigentes_hash_id_idx
    ON impactos_raw (lower(btrim(hash_largo)), id)
    WHERE eliminado_en IS NULL;

CREATE INDEX IF NOT EXISTS perfiles_glaukov_id_grupo_lower_idx
    ON perfiles_glaukov (lower(btrim(id_grupo)));

CREATE INDEX IF NOT EXISTS tasas_glaukov_created_id_idx
    ON tasas_glaukov (created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS comprobantes_liq_socio1_lower_idx
    ON comprobantes_liq (lower(btrim(socio_1)));

CREATE INDEX IF NOT EXISTS comprobantes_liq_socio2_lower_idx
    ON comprobantes_liq (lower(btrim(socio_2)));
