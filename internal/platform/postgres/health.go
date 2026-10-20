package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pinger interface {
	Ping(context.Context) error
}

// Ping bounds the healthcheck so /health cannot outlive the orchestrator probe.
func Ping(ctx context.Context, pool *pgxpool.Pool, timeout time.Duration) error {
	if pool == nil {
		return errors.New("pool de PostgreSQL nil")
	}
	return ping(ctx, pool, timeout)
}

func ping(ctx context.Context, target pinger, timeout time.Duration) error {
	if timeout <= 0 {
		return errors.New("timeout de healthcheck debe ser positivo")
	}
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return target.Ping(pingCtx)
}
