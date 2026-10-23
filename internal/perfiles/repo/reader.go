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
//
// Antes de devolverlos, sanea las filas históricas que aún guardan monedas sin
// tipo/polar_D/polar_P. La escritura se hace después de cerrar el cursor para no
// mantener abierta la conexión de lectura mientras se ejecuta el UPDATE.
func (r *Reader) Listar(ctx context.Context) ([]model.Perfil, error) {
	rows, err := r.db.Query(ctx, ListarSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type leido struct {
		perfil     model.Perfil
		conflictos []model.Conflicto
		crudo      []byte
	}
	leidos := make([]leido, 0)
	for rows.Next() {
		perfil, conflictos, crudo, err := scanPerfil(rows)
		if err != nil {
			return nil, err
		}
		leidos = append(leidos, leido{perfil: perfil, conflictos: conflictos, crudo: crudo})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	perfiles := make([]model.Perfil, 0, len(leidos))
	for _, item := range leidos {
		advertirConflictos(r.log, item.perfil.Nombre, item.conflictos)
		r.normalizarMonedas(ctx, item.perfil, item.crudo)
		perfiles = append(perfiles, item.perfil)
	}
	return perfiles, nil
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
// La fila también pasa por el saneamiento para que Comprobantes y Reportes vean
// el mismo contrato canónico aunque nadie haya abierto el panel de Perfiles.
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
	perfil, conflictos, crudo, err := scanPerfil(rows)
	if err != nil {
		return model.Perfil{}, err
	}
	rows.Close()
	advertirConflictos(r.log, perfil.Nombre, conflictos)
	r.normalizarMonedas(ctx, perfil, crudo)
	return perfil, nil
}
