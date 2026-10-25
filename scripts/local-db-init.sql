-- Solo para el stack local de docker-compose. No se ejecuta en producción.
-- Crea el mínimo necesario para que smoke.ps1 valide monitor y filtros.
-- Para datos reales, restaurar el dump de PostgreSQL y no usar este seed.

BEGIN;

CREATE TABLE IF NOT EXISTS tasas_glaukov (
    id         BIGSERIAL PRIMARY KEY,
    id_tasa    TEXT,
    tasas      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_tasas_glaukov_id_tasa
    ON tasas_glaukov (lower(btrim(id_tasa)))
    WHERE id_tasa IS NOT NULL;

CREATE TABLE IF NOT EXISTS perfiles_glaukov (
    id            BIGSERIAL PRIMARY KEY,
    id_grupo      TEXT NOT NULL DEFAULT '',
    nombre        TEXT NOT NULL,
    rol           TEXT NOT NULL DEFAULT 'SOCIO',
    moneda_base   TEXT NOT NULL DEFAULT 'USDT',
    saldo_inicial NUMERIC,
    mostrar       JSONB NOT NULL DEFAULT '{}'::jsonb,
    monedas       JSONB NOT NULL DEFAULT '{}'::jsonb,
    herencia      BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_perfiles_glaukov_nombre_local
    ON perfiles_glaukov (UPPER(BTRIM(nombre)));

CREATE TABLE IF NOT EXISTS comprobantes_raw (
    hash_largo   TEXT PRIMARY KEY,
    creado_en    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    monto        NUMERIC,
    moneda       TEXT,
    banco        TEXT,
    titular      TEXT,
    referencia   TEXT,
    procesado_ia BOOLEAN NOT NULL DEFAULT FALSE,
    url_r2       TEXT,
    instancia    TEXT NOT NULL DEFAULT 'glauk'
);

CREATE TABLE IF NOT EXISTS comprobantes_liq (
    hash_largo TEXT PRIMARY KEY,
    socio_1    TEXT,
    tipo_op1   TEXT,
    monto_1    NUMERIC,
    tasa_1     NUMERIC,
    me1        NUMERIC,
    socio_2    TEXT,
    tipo_op2   TEXT,
    monto_2    NUMERIC,
    tasa_2     NUMERIC,
    me2        NUMERIC,
    lote_tasa  TEXT
);

CREATE TABLE IF NOT EXISTS impactos_raw (
    id            BIGSERIAL PRIMARY KEY,
    hash_largo    TEXT,
    hash_corto    TEXT,
    url_imagen    TEXT,
    usuario_raw   TEXT,
    grupo_raw     TEXT,
    caption       TEXT,
    timestamp_msg BIGINT,
    instancia     TEXT NOT NULL DEFAULT 'glauk',
    eliminado_en  TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_impactos_raw_dedupe
    ON impactos_raw (
        COALESCE(lower(btrim(hash_largo)), ''),
        COALESCE(usuario_raw, ''),
        COALESCE(grupo_raw, ''),
        COALESCE(timestamp_msg, 0)
    )
    WHERE eliminado_en IS NULL;

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

INSERT INTO tasas_glaukov (id_tasa, tasas)
VALUES ('T001', '{"USD":1,"USDT":1,"COP":4000,"ARS":1250.5,"PYUSD":1.2}'::jsonb)
ON CONFLICT DO NOTHING;

INSERT INTO perfiles_glaukov (id_grupo, nombre, rol, moneda_base)
VALUES
    ('120363000000000001@g.us', 'GENERAL', 'ADMIN', 'USDT'),
    ('120363000000000002@g.us', 'FUNDABLOCK', 'SOCIO', 'USDT')
ON CONFLICT DO NOTHING;

INSERT INTO comprobantes_raw (hash_largo, monto, moneda, banco, referencia)
VALUES ('local0001', 100000, 'COP', 'LOCAL', 'LOCAL-001')
ON CONFLICT DO NOTHING;

INSERT INTO impactos_raw (hash_largo, hash_corto, instancia)
VALUES ('local0001', 'LOCAL01', 'glauk');

COMMIT;
