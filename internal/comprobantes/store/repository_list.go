// Package store es la única puerta de Comprobantes a PostgreSQL.
//
// No decide reglas financieras: recibe filtros ya normalizados, ejecuta SQL y
// devuelve filas o snapshots. Las transacciones se abren con platform/postgres.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"glauk/internal/comprobantes/calc"
	"glauk/internal/comprobantes/domain"
)

// DB es el contrato mínimo que el store necesita de pgx.
type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

var _ DB = (*pgxpool.Pool)(nil)

// Repository implementa lecturas, mutaciones y el puerto de reportes.
type Repository struct {
	db DB
}

// New construye el repositorio con un pool real o un doble de pruebas.
func New(db DB) *Repository {
	return &Repository{db: db}
}

// ListRows ejecuta la consulta de auditoría y devuelve filas crudas.
func (r *Repository) ListRows(ctx context.Context, filters domain.ListFilters) ([]calc.SourceRow, error) {
	query := ListSQL + orderClause(filters.OrderAsc())
	rows, err := r.db.Query(ctx, query,
		filters.Socio, filters.Rol, filters.FechaInicio, filters.FechaFin, filters.Hash,
	)
	if err != nil {
		return nil, fmt.Errorf("consultar comprobantes: %w", err)
	}
	items, err := ScanRows(rows)
	if err != nil {
		return nil, fmt.Errorf("leer comprobantes: %w", err)
	}
	return items, nil
}
