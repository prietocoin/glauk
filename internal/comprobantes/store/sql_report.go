package store

const reportRolesSQL = `
SELECT DISTINCT UPPER(TRIM(rol)) AS rol
FROM perfiles_glaukov
WHERE rol IS NOT NULL AND TRIM(rol) <> ''
ORDER BY rol ASC`

const reportEntitiesSQL = `
SELECT DISTINCT nombre
FROM perfiles_glaukov
WHERE nombre IS NOT NULL AND TRIM(nombre) <> ''
  AND ($1::text = '' OR UPPER(TRIM(rol)) = UPPER(TRIM($1::text)))
ORDER BY nombre ASC`

const reportHashesSQL = `
SELECT hash FROM (
    SELECT COALESCE(NULLIF(TRIM(i.hash_corto), ''), SUBSTRING(c.hash_largo FROM 1 FOR 7)) AS hash,
           COALESCE(c.creado_en, NOW()) AS fecha
    FROM comprobantes_raw c
    LEFT JOIN LATERAL (
        SELECT hash_corto FROM impactos_raw
        WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM(c.hash_largo))
          AND eliminado_en IS NULL AND hash_corto IS NOT NULL AND TRIM(hash_corto) <> ''
        ORDER BY id ASC LIMIT 1
    ) i ON TRUE
    WHERE c.hash_largo IS NOT NULL AND TRIM(c.hash_largo) <> ''
) candidates
GROUP BY hash
ORDER BY MAX(fecha) DESC, hash ASC
LIMIT 100`

const reportFrom = `
FROM comprobantes_raw c
LEFT JOIN comprobantes_liq l ON LOWER(TRIM(l.hash_largo)) = LOWER(TRIM(c.hash_largo))
LEFT JOIN LATERAL (
    SELECT hash_corto FROM impactos_raw
    WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM(c.hash_largo))
      AND eliminado_en IS NULL AND hash_corto IS NOT NULL AND TRIM(hash_corto) <> ''
    ORDER BY id ASC LIMIT 1
) i ON TRUE
LEFT JOIN perfiles_glaukov p
    ON UPPER(TRIM(p.nombre)) = UPPER(TRIM(COALESCE(l.socio_1, 'GENERAL')))
WHERE ($1::text = '' OR UPPER(TRIM(COALESCE(p.rol, 'SOCIO'))) = UPPER(TRIM($1::text)))
  AND ($2::text = '' OR UPPER(TRIM(COALESCE(l.socio_1, 'GENERAL'))) = UPPER(TRIM($2::text)))
  AND ($3::text = '' OR c.hash_largo ILIKE '%' || $3::text || '%'
       OR COALESCE(i.hash_corto, '') ILIKE '%' || $3::text || '%')
  AND ($4::timestamptz IS NULL OR c.creado_en >= $4::timestamptz)
  AND ($5::timestamptz IS NULL OR c.creado_en < ($5::timestamptz + INTERVAL '1 day'))`

const reportMovementsSQL = `
SELECT c.hash_largo,
       COALESCE(i.hash_corto, SUBSTRING(c.hash_largo FROM 1 FOR 7)) AS hash_corto,
       COALESCE(c.creado_en, NOW()) AS fecha,
       COALESCE(l.socio_1, 'GENERAL') AS socio,
       COALESCE(p.rol, 'SOCIO') AS rol,
       COALESCE(UPPER(c.moneda), 'USDT') AS moneda,
       ABS(COALESCE(c.monto, 0)) AS monto,
       COALESCE(l.tipo_op1, 'D-USDT') AS tipo_op,
       COALESCE(c.banco, '-') AS banco,
       COALESCE(c.referencia, '-') AS referencia,
       CASE WHEN l.hash_largo IS NULL THEN 'PENDIENTE' ELSE 'LIQUIDADO' END AS estado` +
	reportFrom + `
ORDER BY fecha `

const reportSummarySQL = `
SELECT UPPER(COALESCE(c.moneda, 'USDT')) AS moneda,
       COUNT(*)::int AS total,
       COALESCE(SUM(ABS(COALESCE(c.monto, 0))), 0)::float8 AS monto,
       MIN(COALESCE(c.creado_en, NOW())) AS desde,
       MAX(COALESCE(c.creado_en, NOW())) AS hasta` +
	reportFrom + `
GROUP BY 1
ORDER BY 1`
