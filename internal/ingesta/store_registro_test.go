package ingesta

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

func TestInsertImpactoUsaClaveUnica(t *testing.T) {
	if strings.Contains(insertImpactoSeguroSQL, "NOT EXISTS") {
		t.Fatal("el INSERT no debe depender de NOT EXISTS")
	}
	if !strings.Contains(insertImpactoSeguroSQL, "ON CONFLICT") {
		t.Fatal("el INSERT debe usar ON CONFLICT")
	}
	if !strings.Contains(insertImpactoSeguroSQL, "eliminado_en IS NULL") {
		t.Fatal("la clave parcial debe excluir impactos eliminados")
	}
}

func TestRegistrarDedupeAtomicoYNormalizaHash(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("mock: %v", err)
	}
	t.Cleanup(mock.Close)

	event := Event{
		HashLargo: " ABC ", HashCorto: "ABC", Instancia: "glauk",
		Usuario: "u", Grupo: "g", Timestamp: 1_700_000_000,
	}

	mock.ExpectExec(insertImpactoSeguroSQL).
		WithArgs("abc", "ABC", "glauk", "u", "g", "", "", int64(1_700_000_000), "RECIBIDO").
		WillReturnResult(pgconn.NewCommandTag("INSERT 0 1"))
	mock.ExpectQuery(insertRegistroSQL).
		WithArgs("abc", "ABC", "g", "u", "", "", int64(1_700_000_000), "glauk", "abc").
		WillReturnRows(pgxmock.NewRows([]string{"conteo", "estado"}))
	mock.ExpectQuery(updateRegistroParSQL).
		WithArgs("abc", "abc").
		WillReturnRows(pgxmock.NewRows([]string{"conteo", "estado"}))
	mock.ExpectQuery(selectRegistroSQL).
		WithArgs("abc").
		WillReturnRows(pgxmock.NewRows([]string{"conteo", "estado"}).
			AddRow(2, "PROCESADO"))

	registro, err := NewRepository(mock, testLogger()).Registrar(context.Background(), event)
	if err != nil {
		t.Fatalf("registrar: %v", err)
	}
	if registro.Conteo != 2 || registro.Estado != EstadoProcesado || registro.Completo {
		t.Fatalf("registro inesperado: %+v", registro)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("consultas pendientes: %v", err)
	}
}
