-- P-02: deduplicación atómica de impactos_raw.
--
-- El pipeline anterior comprobaba con NOT EXISTS y luego insertaba. Dos
-- entregas HTTP paralelas podían pasar ambas la comprobación e insertar el
-- mismo impacto. Este índice crea la clave de identidad del binomio y permite
-- que el INSERT use ON CONFLICT DO NOTHING.
--
-- Los duplicados históricos no se borran: se marcan con eliminado_en para
-- conservar la evidencia y dejarlos fuera del índice parcial.
--
-- Aplicar en ventana de mantenimiento y con backup verificado. Si la tabla es
-- grande, CREATE UNIQUE INDEX toma un lock breve.

WITH ranked AS (
    SELECT id,
           row_number() OVER (
               PARTITION BY COALESCE(lower(btrim(hash_largo)), ''),
                            COALESCE(usuario_raw, ''),
                            COALESCE(grupo_raw, ''),
                            COALESCE(timestamp_msg, 0)
               ORDER BY id
           ) AS rn
    FROM impactos_raw
    WHERE eliminado_en IS NULL
)
UPDATE impactos_raw AS i
SET eliminado_en = NOW()
FROM ranked AS r
WHERE i.id = r.id
  AND r.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS ux_impactos_raw_dedupe
    ON impactos_raw (
        COALESCE(lower(btrim(hash_largo)), ''),
        COALESCE(usuario_raw, ''),
        COALESCE(grupo_raw, ''),
        COALESCE(timestamp_msg, 0)
    )
    WHERE eliminado_en IS NULL;
