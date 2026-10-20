package repo

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"glauk/internal/perfiles/model"
)

// Reader ejecuta las lecturas del directorio. No conoce reglas de negocio: solo
// respeta el contrato de columnas y loguea los conflictos que el codec detectó.
type Reader struct {
	db  Querier
	log *slog.Logger
}

// NewReader construye el lector sobre un Querier de platform/postgres.
func NewReader(db Querier, log *slog.Logger) *Reader {
	if log == nil {
		log = slog.Default()
	}
	return &Reader{db: db, log: log}
}

// Listar devuelve todos los socios ordenados por nombre.
func (r *Reader) Listar(ctx context.Context) ([]model.Perfil, error) {
	rows, err := r.db.Query(ctx, ListarSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	perfiles := make([]model.Perfil, 0)
	for rows.Next() {
		perfil, conflictos, err := scanPerfil(rows)
		if err != nil {
			return nil, err
		}
		advertirConflictos(r.log, perfil.Nombre, conflictos)
		perfiles = append(perfiles, perfil)
	}
	return perfiles, rows.Err()
}

// Activos devuelve los nombres con mostrar.tasas=true, ordenados por nombre.
func (r *Reader) Activos(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, ActivosSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	nombres := make([]string, 0)
	for rows.Next() {
		var nombre string
		if err := rows.Scan(&nombre); err != nil {
			return nil, err
		}
		nombres = append(nombres, nombre)
	}
	return nombres, rows.Err()
}

// PorNombre devuelve una fila o pgx.ErrNoRows. La comparación la hace PostgreSQL
// con UPPER(BTRIM(...)) para respetar la identidad canónica del índice único.
func (r *Reader) PorNombre(ctx context.Context, nombre string) (model.Perfil, error) {
	rows, err := r.db.Query(ctx, PorNombreSQL, nombre)
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
	perfil, conflictos, err := scanPerfil(rows)
	if err != nil {
		return model.Perfil{}, err
	}
	advertirConflictos(r.log, perfil.Nombre, conflictos)
	return perfil, nil
}
