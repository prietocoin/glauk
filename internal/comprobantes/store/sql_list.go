package store

// ListSQL es la consulta de auditoría legacy. Devuelve como máximo 100 filas.
// El orden se añade con orderClause para no duplicar el cuerpo del CTE.
const ListSQL = `
WITH impactos_ordenados AS (
    SELECT hash_largo, hash_corto, url_imagen, usuario_raw, grupo_raw, caption, timestamp_msg,
           ROW_NUMBER() OVER (PARTITION BY LOWER(TRIM(hash_largo)) ORDER BY id ASC) AS num_impacto
    FROM impactos_raw
    WHERE LOWER(TRIM(instancia)) IN ('glauk', 'jairo') AND eliminado_en IS NULL
), datos AS (
    SELECT c.hash_largo,
           COALESCE(i1.hash_corto, SUBSTRING(c.hash_largo FROM 1 FOR 7)) AS hash_corto,
           COALESCE(c.creado_en, CASE WHEN i1.timestamp_msg > 0 THEN to_timestamp(i1.timestamp_msg) END, NOW()) AS fecha_hora_comprobante,
           i1.timestamp_msg,
           COALESCE(c.monto, 0) AS monto,
           COALESCE(UPPER(c.moneda), 'USDT') AS moneda,
           COALESCE(c.banco, '-') AS banco,
           COALESCE(c.titular, '-') AS titular,
           COALESCE(c.referencia, '-') AS referencia,
           COALESCE(c.procesado_ia, FALSE) AS procesado_ia,
           COALESCE(c.url_r2, i1.url_imagen, '') AS url_imagen,
           i1.caption,
           (l.hash_largo IS NOT NULL) AS esta_liquidado,
           l.socio_1, l.tipo_op1, l.monto_1, l.tasa_1, l.me1,
           l.socio_2, l.tipo_op2, l.monto_2, l.tasa_2, l.me2, l.lote_tasa,
           (SELECT t.id_tasa FROM tasas_glaukov t
             WHERE t.created_at <= COALESCE(c.creado_en,
                   CASE WHEN i1.timestamp_msg > 0 THEN to_timestamp(i1.timestamp_msg) END, NOW())
             ORDER BY t.created_at DESC, t.id DESC LIMIT 1) AS lote_tasa_historico,
           tj.tasas AS tasas_lote,
           COALESCE(n1.moneda_base, 'USDT') AS moneda_socio_1,
           COALESCE(n2.moneda_base, 'USDT') AS moneda_socio_2,
           to_jsonb(n1) AS socio1_row, to_jsonb(n2) AS socio2_row,
           COALESCE(n_grupo1.nombre, n_user1.nombre, 'GENERAL') AS fb_socio_1,
           COALESCE(n_grupo2.nombre, n_user2.nombre) AS fb_socio_2,
           COALESCE(n1.rol, 'SOCIO') AS rol_socio_1,
           COALESCE(n2.rol, 'SOCIO') AS rol_socio_2
    FROM comprobantes_raw c
    LEFT JOIN comprobantes_liq l ON LOWER(TRIM(l.hash_largo)) = LOWER(TRIM(c.hash_largo))
    LEFT JOIN impactos_ordenados i1 ON LOWER(TRIM(i1.hash_largo)) = LOWER(TRIM(c.hash_largo)) AND i1.num_impacto = 1
    LEFT JOIN impactos_ordenados i2 ON LOWER(TRIM(i2.hash_largo)) = LOWER(TRIM(c.hash_largo)) AND i2.num_impacto = 2
    LEFT JOIN perfiles_glaukov n_grupo1 ON i1.grupo_raw IS NOT NULL AND TRIM(i1.grupo_raw) <> ''
        AND LOWER(TRIM(n_grupo1.id_grupo)) = LOWER(TRIM(i1.grupo_raw))
    LEFT JOIN perfiles_glaukov n_user1 ON i1.usuario_raw IS NOT NULL AND TRIM(i1.usuario_raw) <> ''
        AND LOWER(TRIM(n_user1.id_grupo)) = LOWER(TRIM(i1.usuario_raw))
    LEFT JOIN perfiles_glaukov n_grupo2 ON i2.grupo_raw IS NOT NULL AND TRIM(i2.grupo_raw) <> ''
        AND LOWER(TRIM(n_grupo2.id_grupo)) = LOWER(TRIM(i2.grupo_raw))
    LEFT JOIN perfiles_glaukov n_user2 ON i2.usuario_raw IS NOT NULL AND TRIM(i2.usuario_raw) <> ''
        AND LOWER(TRIM(n_user2.id_grupo)) = LOWER(TRIM(i2.usuario_raw))
    LEFT JOIN perfiles_glaukov n1 ON UPPER(TRIM(n1.nombre)) = UPPER(TRIM(COALESCE(l.socio_1, n_grupo1.nombre, n_user1.nombre)))
    LEFT JOIN perfiles_glaukov n2 ON UPPER(TRIM(n2.nombre)) = UPPER(TRIM(COALESCE(l.socio_2, n_grupo2.nombre, n_user2.nombre)))
    LEFT JOIN tasas_glaukov tj ON tj.id_tasa = COALESCE(l.lote_tasa, (
        SELECT t.id_tasa FROM tasas_glaukov t
        WHERE t.created_at <= COALESCE(c.creado_en,
              CASE WHEN i1.timestamp_msg > 0 THEN to_timestamp(i1.timestamp_msg) END, NOW())
        ORDER BY t.created_at DESC, t.id DESC LIMIT 1))
    WHERE LOWER(TRIM(c.instancia)) IN ('glauk', 'jairo')
      AND ($5::text = '' OR c.hash_largo ILIKE '%' || $5::text || '%' OR c.referencia ILIKE '%' || $5::text || '%')
      AND (NULLIF($3::text, '') IS NULL OR c.creado_en >= NULLIF($3::text, '')::date)
      AND (NULLIF($4::text, '') IS NULL OR c.creado_en <= (NULLIF($4::text, '')::date + INTERVAL '1 day' - INTERVAL '1 second'))
)
SELECT * FROM datos
WHERE ($1::text = '' OR UPPER(TRIM(rol_socio_1)) = UPPER(TRIM($1::text)) OR UPPER(TRIM(rol_socio_2)) = UPPER(TRIM($1::text)))
  AND ($2::text = '' OR
       UPPER(TRIM(CASE WHEN esta_liquidado THEN socio_1 ELSE fb_socio_1 END)) = UPPER(TRIM($2::text)) OR
       UPPER(TRIM(CASE WHEN esta_liquidado THEN socio_2 ELSE fb_socio_2 END)) = UPPER(TRIM($2::text)))
`

func orderClause(ascending bool) string {
	if ascending {
		return "\nORDER BY fecha_hora_comprobante ASC\nLIMIT 100"
	}
	return "\nORDER BY fecha_hora_comprobante DESC\nLIMIT 100"
}
