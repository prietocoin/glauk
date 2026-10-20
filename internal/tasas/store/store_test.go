package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

// recentSQL es la consulta del lote vigente, escrita literal: un cambio de orden o de
// límite en el paquete debe romper estas pruebas en vez de acompañarlo. Las demás
// consultas van literales dentro de su caso, que es donde se usan.
const recentSQL = `SELECT id_tasa, tasas FROM tasas_glaukov ORDER BY id DESC LIMIT 2`

// columnasLote son las columnas que devuelve la lectura del lote vigente.
var columnasLote = []string{"id_tasa", "tasas"}

// newMock crea el doble del pool: compara el SQL exacto y no toca una base real.
func newMock(t *testing.T) pgxmock.PgxPoolIface {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("no se pudo crear el doble del pool: %v", err)
	}
	t.Cleanup(mock.Close)
	return mock
}

// ptr escribe columnas anulables con la forma que entrega el driver: un puntero.
func ptr[T any](valor T) *T { return &valor }

// TestRecentLotes cubre las tres respuestas posibles de la lectura: filas, tabla
// vacía y fallo del motor.
func TestRecentLotes(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(recentSQL).WillReturnRows(pgxmock.NewRows(columnasLote).AddRow(ptr("T060"), []byte(`{"COP":4050}`)).AddRow(nil, nil))
	lotes, err := RecentLotes(context.Background(), mock)
	if err != nil || len(lotes) != 2 || lotes[0].Tasas["COP"] != 4050.0 || lotes[1].IDTasa != "" {
		t.Fatalf("lotes mal traducidos: %+v, %v", lotes, err)
	}
	mock = newMock(t)
	mock.ExpectQuery(recentSQL).WillReturnRows(pgxmock.NewRows(columnasLote))
	if vacio, err := RecentLotes(context.Background(), mock); err != nil || len(vacio) != 0 {
		t.Fatalf("una tabla vacía no es error: %v, %v", vacio, err)
	}
	roto := errors.New("conexión interrumpida")
	mock = newMock(t)
	mock.ExpectQuery(recentSQL).WillReturnRows(pgxmock.NewRows(columnasLote).AddRow(ptr("T060"), nil).RowError(0, roto))
	if _, err := RecentLotes(context.Background(), mock); !errors.Is(err, roto) {
		t.Fatalf("la causa original debe viajar envuelta: %v", err)
	}
}

// TestEscrituraDeLotes cubre el alta y la baja: la fila devuelta y el conteo de filas.
func TestEscrituraDeLotes(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(`INSERT INTO tasas_glaukov (id_tasa, tasas) VALUES ($1, $2) RETURNING id_tasa, tasas`).WithArgs("T061", []byte(`{"COP":4050}`)).WillReturnRows(pgxmock.NewRows(columnasLote).AddRow(ptr("T061"), []byte(`{"COP":4050}`)))
	lote, err := InsertLote(context.Background(), mock, "T061", map[string]any{"COP": 4050})
	if err != nil || lote.IDTasa != "T061" || len(lote.Tasas) != 1 {
		t.Fatalf("InsertLote devolvió %+v, %v", lote, err)
	}
	mock.ExpectExec(`DELETE FROM tasas_glaukov WHERE LOWER(id_tasa) = LOWER($1)`).WithArgs("T060").WillReturnResult(pgconn.NewCommandTag("DELETE 1"))
	if borrado, err := DeleteLote(context.Background(), mock, "T060"); err != nil || !borrado {
		t.Fatalf("DeleteLote debía informar el borrado: %v, %v", borrado, err)
	}
}

func TestNextLoteIDSaneaLosDigitos(t *testing.T) {
	for ultimo, esperado := range map[string]string{"": "T001", "T059": "T060", "T999": "T1000", "T12A3": "T124"} {
		if obtenido := NextLoteID(ultimo); obtenido != esperado {
			t.Errorf("NextLoteID(%q) = %q, se esperaba %q", ultimo, obtenido, esperado)
		}
	}
}

func TestDecodeJSONRechazaTextoInvalido(t *testing.T) {
	if _, err := decodeJSON[map[string]any]([]byte(`{no es json}`), "tasas"); err == nil {
		t.Error("un JSON inválido debe devolver error y no un mapa vacío")
	}
}
