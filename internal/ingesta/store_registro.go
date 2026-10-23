package ingesta

import (
	"context"
	"fmt"
)

// La identidad del impacto añade usuario, grupo y timestamp para que un
// reintento del webhook no cuente dos veces el mismo binomio.
const insertImpactoSeguroSQL = `
INSERT INTO impactos_raw (
    hash_largo, hash_corto, instancia, usuario_raw, grupo_raw,
    nombre_push, caption, timestamp_msg
)
SELECT $1, $2, $3, $4, $5, $6, $7, $8
WHERE NOT EXISTS (
    SELECT 1 FROM impactos_raw
    WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($1))
      AND COALESCE(usuario_raw, '') = COALESCE($4, '')
      AND COALESCE(grupo_raw, '') = COALESCE($5, '')
      AND COALESCE(timestamp_msg, 0) = COALESCE($8, 0)
)`

const upsertRegistroSQL = `
INSERT INTO registros_raw (
    hash_largo, hash_corto, grupo_raw, usuario_raw, nombre_push,
    caption, timestamp_msg, es_imagen, instancia, url_imagen, conteo, estado
)
VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE, $8, NULL, 1, 'RECIBIDO')
ON CONFLICT (hash_largo) DO UPDATE SET
    conteo = registros_raw.conteo + 1,
    hash_corto = COALESCE(NULLIF(EXCLUDED.hash_corto, ''), registros_raw.hash_corto),
    caption = COALESCE(NULLIF(EXCLUDED.caption, ''), registros_raw.caption),
    usuario_raw = COALESCE(NULLIF(EXCLUDED.usuario_raw, ''), registros_raw.usuario_raw),
    grupo_raw = COALESCE(NULLIF(EXCLUDED.grupo_raw, ''), registros_raw.grupo_raw),
    nombre_push = COALESCE(NULLIF(EXCLUDED.nombre_push, ''), registros_raw.nombre_push)
RETURNING conteo, estado`

const countImpactosSQL = `
SELECT COUNT(*) FROM impactos_raw
WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($1))
  AND eliminado_en IS NULL`

const claimJobSQL = `
UPDATE registros_raw SET estado = 'EN_COLA'
WHERE hash_largo = $1 AND estado IN ('RECIBIDO', 'FALLO')`

const marcarEstadoSQL = `UPDATE registros_raw SET estado = $2 WHERE hash_largo = $1`

// Registrar es idempotente por impacto: inserta la fila de origen y devuelve
// cuántos impactos distintos existen para el hash. El upsert de registros_raw
// se conserva por compatibilidad con el panel de remitHub.
func (r *Repository) Registrar(ctx context.Context, event Event) (Registro, error) {
	if _, err := r.db.Exec(ctx, insertImpactoSeguroSQL,
		event.HashLargo, event.HashCorto, event.Instancia, event.Usuario,
		event.Grupo, event.NombrePush, event.Caption, event.Timestamp); err != nil {
		return Registro{}, fmt.Errorf("ingesta: guardar impacto: %w", err)
	}

	var registro Registro
	var estado string
	err := r.db.QueryRow(ctx, upsertRegistroSQL,
		event.HashLargo, event.HashCorto, event.Grupo, event.Usuario,
		event.NombrePush, event.Caption, event.Timestamp, event.Instancia).
		Scan(&registro.Conteo, &estado)
	if err != nil {
		return Registro{}, fmt.Errorf("ingesta: guardar registro: %w", err)
	}
	registro.Estado = Estado(estado)

	if err := r.db.QueryRow(ctx, countImpactosSQL, event.HashLargo).
		Scan(&registro.Conteo); err != nil {
		return Registro{}, fmt.Errorf("ingesta: contar impactos: %w", err)
	}
	return registro, nil
}

// Claim evita encolar dos veces el mismo binomio: solo la transición desde
// RECIBIDO o FALLO devuelve true.
func (r *Repository) Claim(ctx context.Context, hash string) (bool, error) {
	tag, err := r.db.Exec(ctx, claimJobSQL, hash)
	if err != nil {
		return false, fmt.Errorf("ingesta: reclamar trabajo: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// MarcarEstado cierra el ciclo del registro; en FALLO el operador puede
// reenviar el webhook y Claim lo volverá a encolar.
func (r *Repository) MarcarEstado(ctx context.Context, hash string, estado Estado) error {
	if _, err := r.db.Exec(ctx, marcarEstadoSQL, hash, string(estado)); err != nil {
		return fmt.Errorf("ingesta: marcar %s: %w", estado, err)
	}
	return nil
}
