package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"

	"glauk/internal/comprobantes/domain"
)

func newMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	t.Cleanup(mock.Close)
	return mock
}

func TestDeleteMarksImpactosInSameTransaction(t *testing.T) {
	mock := newMock(t)
	mock.ExpectBeginTx(pgx.TxOptions{})
	mock.ExpectQuery(lockComprobanteSQL).WithArgs("HASH").
		WillReturnRows(pgxmock.NewRows([]string{"monto", "moneda", "banco", "referencia", "titular", "caption"}).
			AddRow(float64(0), nil, nil, nil, nil, ""))
	mock.ExpectExec(deleteLiquidacionSQL).WithArgs("HASH").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectExec(markImpactosSQL).WithArgs("HASH").
		WillReturnResult(pgxmock.NewResult("UPDATE", 2))
	mock.ExpectExec(deleteComprobanteSQL).WithArgs("HASH").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectCommit()
	mock.ExpectRollback().WillReturnError(pgx.ErrTxClosed)

	repo := New(mock)
	err := repo.InTx(context.Background(), func(tx Tx) error {
		return tx.DeleteComprobante(context.Background(), "HASH")
	})
	if err != nil {
		t.Fatalf("InTx: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas: %v", err)
	}
}

func TestDeleteRollsBackWhenMarkImpactosFails(t *testing.T) {
	mock := newMock(t)
	mock.ExpectBeginTx(pgx.TxOptions{})
	mock.ExpectQuery(lockComprobanteSQL).WithArgs("HASH").
		WillReturnRows(pgxmock.NewRows([]string{"monto", "moneda", "banco", "referencia", "titular", "caption"}).
			AddRow(float64(0), nil, nil, nil, nil, ""))
	mock.ExpectExec(deleteLiquidacionSQL).WithArgs("HASH").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectExec(markImpactosSQL).WithArgs("HASH").
		WillReturnError(context.DeadlineExceeded)
	mock.ExpectRollback()

	repo := New(mock)
	err := repo.InTx(context.Background(), func(tx Tx) error {
		return tx.DeleteComprobante(context.Background(), "HASH")
	})
	if err == nil {
		t.Fatal("se esperaba error al marcar impactos")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas: %v", err)
	}
}

func TestUpsertLiquidacionPreservesFrozenValues(t *testing.T) {
	mock := newMock(t)
	socio2 := "SOCIO_B"
	liq := domain.Liquidacion{
		HashLargo: "HASH", Socio1: "SOCIO_A", TipoOp1: "D-COP",
		Monto1: -100, Tasa1: 4000, ME1: -100,
		Socio2: &socio2, TipoOp2: "D-COP", Monto2: 100,
		Tasa2: 4000, ME2: 100, LoteTasa: "T052",
	}
	mock.ExpectExec(upsertLiquidacionSQL).WithArgs(
		"HASH", "SOCIO_A", "D-COP", -100.0, 4000.0, -100.0,
		&socio2, "D-COP", 100.0, 4000.0, 100.0, "T052",
	).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	if err := New(mock).Liquidar(context.Background(), liq); err != nil {
		t.Fatalf("Liquidar: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas: %v", err)
	}
}

func TestUpsertLiquidacionReturnsNotFoundWhenRawMissing(t *testing.T) {
	mock := newMock(t)
	liq := domain.Liquidacion{HashLargo: "HUERFANO", Socio1: "SOCIO_A", LoteTasa: "T052"}
	mock.ExpectExec(upsertLiquidacionSQL).WithArgs(
		"HUERFANO", "SOCIO_A", "", 0.0, 0.0, 0.0,
		(*string)(nil), "", 0.0, 0.0, 0.0, "T052",
	).WillReturnResult(pgxmock.NewResult("INSERT", 0))

	err := New(mock).Liquidar(context.Background(), liq)
	if !errors.Is(err, domain.ErrComprobanteNotFound) {
		t.Fatalf("error = %v, esperado ErrComprobanteNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas: %v", err)
	}
}
