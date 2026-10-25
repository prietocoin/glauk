package ingesta

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// insertImpactoSeguroSQL usa un marcador distinto por columna y por comparación
// del NOT EXISTS. Los casts evitan que PostgreSQL deduzca tipos inconsistentes
// (SQLSTATE 42P08) al reutilizar parámetros en contextos varchar/text/bigint.
//
// La comparación de impactos_raw sigue siendo case-insensitive porque esa tabla
// conserva evidencia histórica en mayúsculas. registros_raw se sanea al arrancar
// y sus búsquedas usan igualdad exacta.
const insertImpactoSeguroSQL = `
INSERT INTO impactos_raw (
    hash_largo, hash_corto, instancia, usuario_raw, grupo_raw,
    nombre_push, caption, timestamp_msg, estado, created_at
)
SELECT
    $1::varchar, $2::varchar, $3::varchar, $4::varchar, $5::varchar,
    $6::varchar, $7::text, $8::bigint, $9::varchar, NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM impactos_raw
    WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($10::varchar))
      AND COALESCE(usuario_raw, '') = COALESCE($11::varchar, '')
      AND COALESCE(grupo_raw, '') = COALESCE($12::varchar, '')
      AND COALESCE(timestamp_msg, 0) = COALESCE($13::bigint, 0)
      AND eliminado_en IS NULL
)`

// insertRegistroSQL crea la fila del impacto 1 con conteo = 1. Si la fila ya
// existe, DO NOTHING deja que el flujo decida si toca completar el binomio.
const insertRegistroSQL = `
INSERT INTO registros_raw (
    hash_largo, hash_corto, grupo_raw, usuario_raw, nombre_push,
    caption, timestamp_msg, es_imagen, instancia, url_imagen,
    conteo, estado, hash_imagen
)
VALUES (
    $1::varchar, $2::varchar, $3::varchar, $4::varchar, $5::varchar,
    $6::text, $7::bigint, TRUE, $8::varchar, NULL,
    1, 'RECIBIDO', $9::varchar
)
ON CONFLICT (hash_largo) DO NOTHING
RETURNING COALESCE(conteo, 0), COALESCE(estado, 'RECIBIDO')`

// updateRegistroParSQL completa el binomio usando el segundo impacto real de
// impactos_raw (OFFSET 1), no el payload que reintenta el webhook. Así un
// reintento recupera el estado aunque el UPDATE anterior haya fallado.
const updateRegistroParSQL = `
UPDATE registros_raw AS r
SET conteo = 2,
    usuario_raw_2 = COALESCE(r.usuario_raw_2, s.usuario_raw),
    grupo_raw_2 = COALESCE(r.grupo_raw_2, s.grupo_raw),
    estado = 'PROCESADO',
    hash_imagen = COALESCE(r.hash_imagen, r.hash_largo),
    timestamp_msg = COALESCE(r.timestamp_msg, s.timestamp_msg)
FROM (
    SELECT usuario_raw, grupo_raw, timestamp_msg
    FROM impactos_raw
    WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($1::varchar))
      AND eliminado_en IS NULL
    ORDER BY id ASC
    LIMIT 1 OFFSET 1
) AS s
WHERE r.hash_largo = $2::varchar
  AND COALESCE(r.conteo, 0) < 2
RETURNING r.conteo, COALESCE(r.estado, 'PROCESADO')`

const selectRegistroSQL = `
SELECT COALESCE(conteo, 0), COALESCE(estado, 'RECIBIDO')
FROM registros_raw WHERE hash_largo = $1::varchar`

const claimJobSQL = `
UPDATE registros_raw SET estado = 'EN_COLA'
WHERE hash_largo = $1::varchar AND estado IN ('RECIBIDO', 'FALLO', 'PROCESADO')`

const marcarEstadoSQL = `UPDATE registros_raw SET estado = $2::varchar WHERE hash_largo = $1::varchar`

// Registrar inserta el impacto y mantiene el binomio en registros_raw con el
// esquema real: impacto 1 INSERT conteo=1; impacto 2 UPDATE conteo=2 y
// estado='PROCESADO'. Devuelve Completo solo cuando esta llamada cerró el par.
func (r *Repository) Registrar(ctx context.Context, event Event) (Registro, error) {
	event.HashLargo = normalizeHash(event.HashLargo)
	if _, err := r.db.Exec(ctx, insertImpactoSeguroSQL,
		event.HashLargo, event.HashCorto, event.Instancia, event.Usuario,
		event.Grupo, event.NombrePush, event.Caption, event.Timestamp,
		string(EstadoRecibido),
		event.HashLargo, event.Usuario, event.Grupo, event.Timestamp); err != nil {
		return Registro{}, r.sqlFail("insert_impactos_raw",
			fmt.Errorf("ingesta: guardar impacto: %w", err))
	}

	var registro Registro
	var estado string
	err := r.db.QueryRow(ctx, insertRegistroSQL,
		event.HashLargo, event.HashCorto, event.Grupo, event.Usuario,
		event.NombrePush, event.Caption, event.Timestamp, event.Instancia,
		event.HashLargo).
		Scan(&registro.Conteo, &estado)
	switch {
	case err == nil:
		registro.Estado = Estado(estado)
	case !errors.Is(err, pgx.ErrNoRows):
		return Registro{}, r.sqlFail("insert_registros_raw",
			fmt.Errorf("ingesta: guardar registro: %w", err))
	}

	err = r.db.QueryRow(ctx, updateRegistroParSQL, event.HashLargo, event.HashLargo).
		Scan(&registro.Conteo, &estado)
	if err == nil {
		registro.Estado = Estado(estado)
		registro.Completo = true
		return registro, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Registro{}, r.sqlFail("update_registros_raw_par",
			fmt.Errorf("ingesta: completar binomio: %w", err))
	}
	if registro.Conteo > 0 {
		return registro, nil
	}

	err = r.db.QueryRow(ctx, selectRegistroSQL, event.HashLargo).
		Scan(&registro.Conteo, &estado)
	if err != nil {
		return Registro{}, r.sqlFail("select_registros_raw",
			fmt.Errorf("ingesta: leer registro: %w", err))
	}
	registro.Estado = Estado(estado)
	return registro, nil
}

// Claim evita encolar dos veces el mismo binomio. PROCESADO se acepta porque
// el impacto 2 marca el par como completo antes de encolarlo.
func (r *Repository) Claim(ctx context.Context, hash string) (bool, error) {
	hash = normalizeHash(hash)
	tag, err := r.db.Exec(ctx, claimJobSQL, hash)
	if err != nil {
		return false, r.sqlFail("claim_registros_raw",
			fmt.Errorf("ingesta: reclamar trabajo: %w", err))
	}
	return tag.RowsAffected() > 0, nil
}

// MarcarEstado cierra el ciclo del registro; en FALLO el operador puede
// reenviar el webhook y Claim lo volverá a encolar.
func (r *Repository) MarcarEstado(ctx context.Context, hash string, estado Estado) error {
	hash = normalizeHash(hash)
	if _, err := r.db.Exec(ctx, marcarEstadoSQL, hash, string(estado)); err != nil {
		return r.sqlFail("marcar_estado_registros_raw",
			fmt.Errorf("ingesta: marcar %s: %w", estado, err))
	}
	return nil
}
