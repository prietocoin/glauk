package ingesta

import (
	"context"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"

	"glauk/internal/comprobantes/domain"
)

func liquidacionValida(socio2 string) LiquidacionResult {
	ptr := socio2
	return LiquidacionResult{Snapshot: domain.Liquidacion{
		HashLargo: "HASH", Socio1: "SOCIO_A", TipoOp1: "D-COP",
		Monto1: -100, Tasa1: 4000, ME1: -100,
		Socio2: &ptr, TipoOp2: "D-COP", Monto2: 100, Tasa2: 4000, ME2: 100,
		LoteTasa: "T052",
	}}
}

func TestGuardarLiquidacionRechazaEquivalentesCero(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("mock: %v", err)
	}
	t.Cleanup(mock.Close)

	liq := liquidacionValida("SOCIO_B")
	liq.Snapshot.ME1 = 0
	liq.Snapshot.ME2 = 0

	err = NewRepository(mock, testLogger()).GuardarLiquidacion(context.Background(), liq)
	if err == nil || !strings.Contains(err.Error(), "impacto contable") {
		t.Fatalf("error = %v, esperado rechazo por equivalentes cero", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("no debió escribir en base: %v", err)
	}
}

func TestGuardarLiquidacionExigePerfilesRegistrados(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("mock: %v", err)
	}
	t.Cleanup(mock.Close)

	mock.ExpectQuery(perfilExisteSQL).WithArgs("SOCIO_A").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	err = NewRepository(mock, testLogger()).GuardarLiquidacion(context.Background(),
		liquidacionValida("SOCIO_B"))
	if err == nil || !strings.Contains(err.Error(), "no está registrado") {
		t.Fatalf("error = %v, esperado rechazo de perfil", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas: %v", err)
	}
}

func TestGuardarLiquidacionPersisteSnapshotValido(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("mock: %v", err)
	}
	t.Cleanup(mock.Close)

	mock.ExpectQuery(perfilExisteSQL).WithArgs("SOCIO_A").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(perfilExisteSQL).WithArgs("SOCIO_B").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(upsertLiquidacionSQL).
		WithArgs("hash", "SOCIO_A", "D-COP", -100.0, 4000.0, -100.0,
			"SOCIO_B", "D-COP", 100.0, 4000.0, 100.0, "T052").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	if err := NewRepository(mock, testLogger()).GuardarLiquidacion(context.Background(),
		liquidacionValida("SOCIO_B")); err != nil {
		t.Fatalf("guardar: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas: %v", err)
	}
}
