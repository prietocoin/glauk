package ingesta

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const imagenExistenteSQL = `
SELECT url_imagen FROM impactos_raw
WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($1::varchar))
  AND url_imagen IS NOT NULL AND url_imagen <> ''
  AND eliminado_en IS NULL
ORDER BY id ASC LIMIT 1`

const actualizarImagenSQL = `
UPDATE impactos_raw SET url_imagen = $2::text
WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($1::varchar))
  AND (url_imagen IS NULL OR url_imagen = '')`

const actualizarRegistroImagenSQL = `
UPDATE registros_raw SET url_imagen = $2::text
WHERE hash_largo = $1::varchar AND (url_imagen IS NULL OR url_imagen = '')`

const contextoComprobanteSQL = `
SELECT COALESCE(hash_corto, ''), COALESCE(instancia, ''), COALESCE(url_r2, ''),
       COALESCE(monto, 0), COALESCE(moneda, ''), COALESCE(banco, ''),
       COALESCE(titular, ''), COALESCE(referencia, ''), COALESCE(procesado_ia, FALSE)
FROM comprobantes_raw
WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($1::varchar))
LIMIT 1`

const contextoImpactosSQL = `
SELECT COALESCE(grupo_raw, ''), COALESCE(caption, ''),
       COALESCE(timestamp_msg, 0), COALESCE(url_imagen, '')
FROM impactos_raw
WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($1::varchar))
  AND eliminado_en IS NULL
ORDER BY id ASC`

// ImagenExistente devuelve la URL ya publicada en cualquier impacto, para no
// volver a subir el mismo binario cuando llega el segundo mensaje.
func (r *Repository) ImagenExistente(ctx context.Context, hash string) (string, error) {
	var url string
	err := r.db.QueryRow(ctx, imagenExistenteSQL, hash).Scan(&url)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", r.sqlFail("select_imagen_existente",
			fmt.Errorf("ingesta: leer imagen existente: %w", err))
	}
	return url, nil
}

// ActualizarImagen propaga la URL a los impactos que aún no la tenían y, de
// forma best-effort, al registro que lee el panel.
func (r *Repository) ActualizarImagen(ctx context.Context, hash, url string) error {
	if _, err := r.db.Exec(ctx, actualizarImagenSQL, hash, url); err != nil {
		return r.sqlFail("update_impactos_raw_imagen",
			fmt.Errorf("ingesta: actualizar imagen: %w", err))
	}
	if _, err := r.db.Exec(ctx, actualizarRegistroImagenSQL, hash, url); err != nil {
		_ = r.sqlFail("update_registros_raw_imagen", err)
	}
	return nil
}

// Contexto mezcla la fila de comprobantes_raw (si ya existe) con los metadatos
// de los dos impactos. No es un error que el comprobante todavía no exista.
func (r *Repository) Contexto(ctx context.Context, hash string) (Contexto, error) {
	result := Contexto{HashLargo: hash}
	err := r.db.QueryRow(ctx, contextoComprobanteSQL, hash).Scan(
		&result.HashCorto, &result.Instancia, &result.URLImagen,
		&result.Monto, &result.Moneda, &result.Banco,
		&result.Titular, &result.Referencia, &result.Procesado,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Contexto{}, r.sqlFail("select_comprobantes_raw_contexto",
			fmt.Errorf("ingesta: leer comprobante: %w", err))
	}

	rows, err := r.db.Query(ctx, contextoImpactosSQL, hash)
	if err != nil {
		return Contexto{}, r.sqlFail("select_impactos_raw_contexto",
			fmt.Errorf("ingesta: leer impactos: %w", err))
	}
	defer rows.Close()

	for rows.Next() {
		var grupo, caption, url string
		var timestamp int64
		if err := rows.Scan(&grupo, &caption, &timestamp, &url); err != nil {
			return Contexto{}, r.sqlFail("scan_impactos_raw_contexto",
				fmt.Errorf("ingesta: escanear impacto: %w", err))
		}
		if result.Caption == "" && caption != "" {
			result.Caption = caption
		}
		if result.Timestamp == 0 && timestamp > 0 {
			result.Timestamp = timestamp
		}
		if result.URLImagen == "" && url != "" {
			result.URLImagen = url
		}
		if grupo != "" && !containsString(result.Grupos, grupo) && len(result.Grupos) < 2 {
			result.Grupos = append(result.Grupos, grupo)
		}
	}
	if err := rows.Err(); err != nil {
		return Contexto{}, r.sqlFail("rows_impactos_raw_contexto",
			fmt.Errorf("ingesta: recorrer impactos: %w", err))
	}
	return result, nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
