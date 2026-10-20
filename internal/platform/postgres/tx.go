package postgres

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
)

const rollbackTimeout = 5 * time.Second

// Beginner is the minimal transaction opener shared by *pgxpool.Pool and the
// pgxmock pool used in repository tests.
type Beginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// WithTx runs fn inside a transaction and guarantees rollback on error or
// panic. A rollback after a successful commit returns pgx.ErrTxClosed and is
// deliberately ignored.
func WithTx(ctx context.Context, beginner Beginner, opts pgx.TxOptions,
	fn func(pgx.Tx) error) error {
	if beginner == nil || isNilBeginner(beginner) {
		return fmt.Errorf("pool de PostgreSQL nil")
	}
	return withTx(ctx, beginner, opts, fn)
}

func isNilBeginner(beginner Beginner) bool {
	value := reflect.ValueOf(beginner)
	return value.Kind() == reflect.Ptr && value.IsNil()
}

func withTx(ctx context.Context, beginner Beginner, opts pgx.TxOptions,
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
