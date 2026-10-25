package ingesta

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB es el contrato mínimo de pgx que usa el pipeline. Coincide con el Querier
// de internal/tasas, por lo que el adaptador puede pasar el mismo pool.
type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

var _ DB = (*pgxpool.Pool)(nil)

// Repository concentra todo el SQL de ingesta. No decide reglas financieras:
// guarda lo que el servicio ya normalizó.
type Repository struct {
	db  DB
	log *slog.Logger
}

// NewRepository construye el repositorio con un pool real o un doble.
func NewRepository(db DB, log *slog.Logger) *Repository {
	if log == nil {
		log = slog.Default()
	}
	return &Repository{db: db, log: log}
}

var _ Store = (*Repository)(nil)

// sqlFail deja rastro explícito del query que rompió el webhook. El error se
// devuelve envuelto para que el handler responda 500 con contexto.
func (r *Repository) sqlFail(query string, err error) error {
	if err != nil && r.log != nil {
		r.log.Error("error sql en webhook", "error", err, "query", query)
	}
	return err
}
