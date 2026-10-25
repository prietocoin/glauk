package store

import (
	"context"
	"errors"
	"strings"
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

func TestListSQLRequiresLiquidatedAndPositiveAmount(t *testing.T) {
	sql := strings.Join(strings.Fields(ListSQL), " ")
	for _, want := range []string{
		"l.hash_largo IS NOT NULL",
		"l.socio_2 IS NOT NULL",
		"btrim(l.socio_2) <> ''",
		"UPPER(btrim(l.socio_2)) <> 'GENERAL'",
		"c.monto > 0",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("ListSQL debe exigir binomio 2x: falta %q", want)
		}
	}
}

func TestListRowsMapsSocioToDollarTwo(t *testing.T) {
	mock := newMock(t)
	query := strings.ReplaceAll(ListSQL, "{{PAGE_ORDER}}", orderClause(false, 100, 0))
	query = strings.ReplaceAll(query, "{{OUTER_ORDER}}", outerOrderClause(false))

	// $1 = rol, $2 = socio, $3 = fechaInicio, $4 = fechaFin, $5 = hash.
	// Antes del fix se enviaba Socio en $1 y Rol en $2, así que ?socio=CHASAN
	// filtraba por rol y devolvía 0 filas.
	mock.ExpectQuery(query).
		WithArgs("", "CHASAN", "", "", "").
		WillReturnRows(pgxmock.NewRows(make([]string, 34)))

	items, total, err := New(mock).ListRows(context.Background(), domain.ListFilters{
		Socio: "CHASAN", Limit: 100, Orden: "fecha_desc",
	})
	if err != nil {
		t.Fatalf("ListRows: %v", err)
	}
	if total != 0 || len(items) != 0 {
		t.Fatalf("resultado inesperado: total=%d items=%d", total, len(items))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas: %v", err)
	}
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
