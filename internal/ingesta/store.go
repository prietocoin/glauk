package ingesta

import (
	"context"
	"sync"

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
	db DB

	mu    sync.Mutex
	final *bool
}

// NewRepository construye el repositorio con un pool real o un doble.
func NewRepository(db DB) *Repository {
	return &Repository{db: db}
}

var _ Store = (*Repository)(nil)

// finalDisponible pregunta una sola vez si la tabla legacy existe. Se cachea
// para no pagar la consulta en cada comprobante y para que una base sin
// comprobantes_final siga procesando sin escribirla.
func (r *Repository) finalDisponible(ctx context.Context) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.final != nil {
		return *r.final
	}
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT to_regclass('comprobantes_final') IS NOT NULL`).Scan(&exists)
	if err != nil {
		return false
	}
	r.final = &exists
	return exists
}
