package store

import (
	"fmt"

	"glauk/internal/comprobantes/domain"
)

// ListSQL es la consulta de auditoría. Primero se resuelven los candidatos de
// comprobantes_raw con solo los dos impactos necesarios para rol/socio, luego
// se pagina y recién entonces se expande la evidencia de presentación y el
// lote de tasas.
//
// El total del universo filtrado se calcula con COUNT(*) OVER() dentro de
// "filtrados", es decir, antes de los joins de presentación. Los marcadores
// {{PAGE_ORDER}} y {{OUTER_ORDER}} los completa repository_list.go para no
// duplicar el cuerpo de la consulta.
const ListSQL = `
WITH candidatos AS (
    SELECT c.hash_largo,
           c.creado_en,
           COALESCE(i1.hash_corto, SUBSTRING(c.hash_largo FROM 1 FOR 7)) AS hash_corto,
           COALESCE(c.creado_en,
                    CASE WHEN i1.timestamp_msg > 0 THEN to_timestamp(i1.timestamp_msg) END,
                    NOW()) AS fecha_hora_comprobante,
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
           COALESCE(n1.moneda_base, 'USDT') AS moneda_socio_1,
           COALESCE(n2.moneda_base, 'USDT') AS moneda_socio_2,
           to_jsonb(n1) AS socio1_row,
           to_jsonb(n2) AS socio2_row,
           COALESCE(n_grupo1.nombre, n_user1.nombre, 'GENERAL') AS fb_socio_1,
           COALESCE(n_grupo2.nombre, n_user2.nombre) AS fb_socio_2,
           COALESCE(n1.rol, 'SOCIO') AS rol_socio_1,
           COALESCE(n2.rol, 'SOCIO') AS rol_socio_2
    FROM comprobantes_raw c
    LEFT JOIN comprobantes_liq l
        ON lower(btrim(l.hash_largo)) = lower(btrim(c.hash_largo))
    LEFT JOIN LATERAL (
        SELECT i.hash_corto, i.url_imagen, i.usuario_raw, i.grupo_raw,
               i.caption, i.timestamp_msg
        FROM impactos_raw i
        WHERE lower(btrim(i.hash_largo)) = lower(btrim(c.hash_largo))
          AND lower(btrim(i.instancia)) IN ('glauk', 'jairo')
          AND i.eliminado_en IS NULL
        ORDER BY i.id ASC
        LIMIT 1
    ) i1 ON TRUE
    LEFT JOIN LATERAL (
        SELECT i.usuario_raw, i.grupo_raw
        FROM impactos_raw i
        WHERE lower(btrim(i.hash_largo)) = lower(btrim(c.hash_largo))
          AND lower(btrim(i.instancia)) IN ('glauk', 'jairo')
          AND i.eliminado_en IS NULL
        ORDER BY i.id ASC
        OFFSET 1 LIMIT 1
    ) i2 ON TRUE
    LEFT JOIN perfiles_glaukov n_grupo1
        ON i1.grupo_raw IS NOT NULL AND TRIM(i1.grupo_raw) <> ''
       AND lower(btrim(n_grupo1.id_grupo)) = lower(btrim(i1.grupo_raw))
    LEFT JOIN perfiles_glaukov n_user1
        ON i1.usuario_raw IS NOT NULL AND TRIM(i1.usuario_raw) <> ''
       AND lower(btrim(n_user1.id_grupo)) = lower(btrim(i1.usuario_raw))
    LEFT JOIN perfiles_glaukov n_grupo2
        ON i2.grupo_raw IS NOT NULL AND TRIM(i2.grupo_raw) <> ''
       AND lower(btrim(n_grupo2.id_grupo)) = lower(btrim(i2.grupo_raw))
    LEFT JOIN perfiles_glaukov n_user2
        ON i2.usuario_raw IS NOT NULL AND TRIM(i2.usuario_raw) <> ''
       AND lower(btrim(n_user2.id_grupo)) = lower(btrim(i2.usuario_raw))
    LEFT JOIN perfiles_glaukov n1
        ON upper(btrim(n1.nombre)) = upper(btrim(COALESCE(l.socio_1, n_grupo1.nombre, n_user1.nombre)))
    LEFT JOIN perfiles_glaukov n2
        ON upper(btrim(n2.nombre)) = upper(btrim(COALESCE(l.socio_2, n_grupo2.nombre, n_user2.nombre)))
    WHERE lower(btrim(c.instancia)) IN ('glauk', 'jairo')
      AND ($5::text = '' OR c.hash_largo ILIKE '%' || $5::text || '%' OR c.referencia ILIKE '%' || $5::text || '%')
      AND (NULLIF($3::text, '') IS NULL OR c.creado_en >= NULLIF($3::text, '')::date)
      AND (NULLIF($4::text, '') IS NULL OR c.creado_en <= (NULLIF($4::text, '')::date + INTERVAL '1 day' - INTERVAL '1 second'))
), filtrados AS (
    SELECT candidatos.*,
           COUNT(*) OVER() AS total_registros
    FROM candidatos
    WHERE ($1::text = '' OR UPPER(TRIM(rol_socio_1)) = UPPER(TRIM($1::text)) OR UPPER(TRIM(rol_socio_2)) = UPPER(TRIM($1::text)))
      AND (
        $2::text = '' OR
        UPPER(TRIM(CASE WHEN esta_liquidado THEN socio_1 ELSE fb_socio_1 END)) = UPPER(TRIM($2::text)) OR
        UPPER(TRIM(CASE WHEN esta_liquidado THEN socio_2 ELSE fb_socio_2 END)) = UPPER(TRIM($2::text))
      )
), pagina AS (
    SELECT *
    FROM filtrados
    {{PAGE_ORDER}}
)
SELECT p.hash_largo,
       p.hash_corto,
       p.fecha_hora_comprobante,
       p.timestamp_msg,
       p.monto,
       p.moneda,
       p.banco,
       p.titular,
       p.referencia,
       p.procesado_ia,
       p.url_imagen,
       p.caption,
       p.esta_liquidado,
       p.socio_1, p.tipo_op1, p.monto_1, p.tasa_1, p.me1,
       p.socio_2, p.tipo_op2, p.monto_2, p.tasa_2, p.me2, p.lote_tasa,
       historico.id_tasa AS lote_tasa_historico,
       tj.tasas AS tasas_lote,
       p.moneda_socio_1,
       p.moneda_socio_2,
       p.socio1_row,
       p.socio2_row,
       p.fb_socio_1,
       p.fb_socio_2,
       p.rol_socio_1,
       p.rol_socio_2,
       p.total_registros
FROM pagina p
LEFT JOIN LATERAL (
    SELECT t.id_tasa
    FROM tasas_glaukov t
    WHERE t.created_at <= COALESCE(p.creado_en,
          CASE WHEN p.timestamp_msg > 0 THEN to_timestamp(p.timestamp_msg) END,
          NOW())
    ORDER BY t.created_at DESC, t.id DESC
    LIMIT 1
) historico ON TRUE
LEFT JOIN tasas_glaukov tj
    ON tj.id_tasa = COALESCE(p.lote_tasa, historico.id_tasa)
{{OUTER_ORDER}}
`

// orderClause añade el orden y la ventana paginada de la CTE "pagina".
// limit/offset llegan ya normalizados, pero se acotan otra vez para que ningún
// llamador pueda saltarse el tope de filas por página.
func orderClause(ascending bool, limit, offset int) string {
	if limit <= 0 {
		limit = domain.DefaultListLimit
	}
	if limit > domain.MaxListLimit {
		limit = domain.MaxListLimit
	}
	if offset < 0 {
		offset = 0
	}
	direction := "DESC"
	if ascending {
		direction = "ASC"
	}
	return fmt.Sprintf("\nORDER BY fecha_hora_comprobante %s, hash_largo %s\nLIMIT %d OFFSET %d",
		direction, direction, limit, offset)
}

// outerOrderClause conserva el orden de la página ya materializada. Sin él,
// PostgreSQL podría devolver las filas de la CTE en cualquier orden.
func outerOrderClause(ascending bool) string {
	direction := "DESC"
	if ascending {
		direction = "ASC"
	}
	return fmt.Sprintf("\nORDER BY p.fecha_hora_comprobante %s, p.hash_largo %s", direction, direction)
}
