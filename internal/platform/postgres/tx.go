package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const rollbackTimeout = 5 * time.Second

type txBeginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// WithTx runs fn inside a transaction and guarantees rollback on error or
// panic. A rollback after a successful commit returns pgx.ErrTxClosed and is
// deliberately ignored.
func WithTx(ctx context.Context, pool *pgxpool.Pool, opts pgx.TxOptions,
	fn func(pgx.Tx) error) error {
	if pool == nil {
		return fmt.Errorf("pool de PostgreSQL nil")
	}
	return withTx(ctx, pool, opts, fn)
}

func withTx(ctx context.Context, beginner txBeginner, opts pgx.TxOptions,
	fn func(pgx.Tx) error) error {
	tx, err := beginner.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("iniciar transacción: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			rollbackTx(ctx, tx)
			panic(r)
		}
		rollbackTx(ctx, tx)
	}()

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("confirmar transacción: %w", err)
	}
	return nil
}

func rollbackTx(ctx context.Context, tx pgx.Tx) {
	rbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
	defer cancel()
	if err := tx.Rollback(rbCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		// El error de negocio o de commit ya se está propagando.
	}
}
