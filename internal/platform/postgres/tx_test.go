package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func newMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	t.Cleanup(mock.Close)
	return mock
}

func TestWithTxCommits(t *testing.T) {
	mock := newMock(t)
	opts := pgx.TxOptions{IsoLevel: pgx.Serializable}
	mock.ExpectBeginTx(opts)
	mock.ExpectCommit()
	mock.ExpectRollback().WillReturnError(pgx.ErrTxClosed)

	called := false
	err := withTx(context.Background(), mock, opts, func(tx pgx.Tx) error {
		called = true
		if tx == nil {
			return errors.New("tx nil")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTx: %v", err)
	}
	if !called {
		t.Fatal("no se ejecutó la función")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas: %v", err)
	}
}

func TestWithTxRollsBackOnError(t *testing.T) {
	mock := newMock(t)
	opts := pgx.TxOptions{}
	mock.ExpectBeginTx(opts)
	mock.ExpectRollback()

	boom := errors.New("boom")
	err := withTx(context.Background(), mock, opts, func(pgx.Tx) error {
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas: %v", err)
	}
}

func TestWithTxRollsBackOnPanic(t *testing.T) {
	mock := newMock(t)
	opts := pgx.TxOptions{}
	mock.ExpectBeginTx(opts)
	mock.ExpectRollback()

	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Fatal("el pánico debía propagarse")
			}
		}()
		_ = withTx(context.Background(), mock, opts, func(pgx.Tx) error {
			panic("boom")
		})
	}()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas: %v", err)
	}
}

func TestWithTxBeginFailure(t *testing.T) {
	mock := newMock(t)
	opts := pgx.TxOptions{}
	boom := errors.New("begin")
	mock.ExpectBeginTx(opts).WillReturnError(boom)

	called := false
	err := withTx(context.Background(), mock, opts, func(pgx.Tx) error {
		called = true
		return nil
	})
	if !errors.Is(err, boom) || called {
		t.Fatalf("err = %v, called = %v", err, called)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas: %v", err)
	}
}

func TestWithTxRejectsNilPool(t *testing.T) {
	err := WithTx(context.Background(), nil, pgx.TxOptions{}, func(pgx.Tx) error {
		return nil
	})
	if err == nil {
		t.Fatal("se esperaba error por pool nil")
	}
}
