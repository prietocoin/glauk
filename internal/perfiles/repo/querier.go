// Package repo es la única puerta de escritura a perfiles_glaukov.
//
// Consume un Querier, nunca abre un pool. El pool concreto llega desde
// platform/postgres en cmd/server; las pruebas inyectan pgxmock o un doble.
package repo

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier es el contrato mínimo que el repositorio necesita del motor.
//
// Se declara localmente para que este dominio no dependa del paquete Tasas ni
// abra un pool propio. Cualquier implementación de pgx que cumpla estas firmas
// sirve: *pgxpool.Pool en producción y pgxmock en tests.
type Querier interface {
	// Query se usa para lecturas y para el RETURNING del upsert.
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	// QueryRow queda disponible para consultas puntuales futuras.
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	// Exec se usa en borrados y operaciones masivas de plantilla.
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// El pool real de platform/postgres satisface el contrato sin adaptador. Esta
// afirmación rompe la compilación si el cliente de pgx cambia la firma.
var _ Querier = (*pgxpool.Pool)(nil)
