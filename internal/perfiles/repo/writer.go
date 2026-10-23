package repo

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"

	"glauk/internal/perfiles/model"
)

// UpsertParams es el contrato ya normalizado por el servicio. El repositorio no
// aplica defaults de negocio: recibe mostrar y monedas listos para persistir.
type UpsertParams struct {
	IDGrupo      string
	Nombre       string
	Rol          string
	MonedaBase   string
	SaldoInicial string
	Mostrar      model.Mostrar
	Monedas      model.Monedas
	Herencia     bool
}

// Writer agrupa las mutaciones individuales del directorio.
type Writer struct {
	db Querier
}

// NewWriter construye el escritor sobre el pool de platform/postgres.
func NewWriter(db Querier) *Writer {
	return &Writer{db: db}
}

// Guardar ejecuta el upsert atómico y devuelve la fila resultante.
func (w *Writer) Guardar(ctx context.Context, params UpsertParams) (model.Perfil, error) {
	mostrar, err := json.Marshal(params.Mostrar)
	if err != nil {
		return model.Perfil{}, err
	}
	monedas, err := encodeMonedas(params.Monedas)
	if err != nil {
		return model.Perfil{}, err
	}
	return w.queryOne(ctx, UpsertSQL,
		params.IDGrupo,
		params.Nombre,
		params.Rol,
		params.MonedaBase,
		params.SaldoInicial,
		mostrar,
		monedas,
		params.Herencia,
	)
}

// Toggle cambia mostrar.tasas y devuelve el perfil con activo recalculado.
func (w *Writer) Toggle(ctx context.Context, nombre string, activo bool) (model.Perfil, error) {
	return w.queryOne(ctx, ToggleSQL, activo, nombre)
}

// Delete elimina un socio y convierte cero filas en pgx.ErrNoRows.
func (w *Writer) Delete(ctx context.Context, nombre string) error {
	tag, err := w.db.Exec(ctx, DeleteSQL, nombre)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (w *Writer) queryOne(ctx context.Context, sql string, args ...any) (model.Perfil, error) {
	rows, err := w.db.Query(ctx, sql, args...)
	if err != nil {
		return model.Perfil{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return model.Perfil{}, err
		}
		return model.Perfil{}, pgx.ErrNoRows
	}
	perfil, _, _, err := scanPerfil(rows)
	return perfil, err
}
