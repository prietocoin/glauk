package repo

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"

	"glauk/internal/perfiles/model"
)

type contiene struct{ textos []string }

func (c contiene) Match(valor any) bool {
	var contenido []byte
	switch actual := valor.(type) {
	case []byte:
		contenido = actual
	case string:
		contenido = []byte(actual)
	default:
		return false
	}
	for _, texto := range c.textos {
		if !bytes.Contains(contenido, []byte(texto)) {
			return false
		}
	}
	return true
}

func TestWriterGuardarSuperset(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	monedas := []byte(`{"ARS":{"activo":true,"porcentaje":{"deposito":2.5,"pago":1.5},"def":"D","deposito":2.5,"dep_op":"+","pago":1.5,"pago_op":"-","tipo":"D","polar_D":"+","polar_P":"-","polaridad":"+"}}`)
	rows := pgxmock.NewRows(columnasPerfil).AddRow(
		"120@g.us", "NUEVO", "SOCIO", "USDT", "10",
		[]byte(`{"tasas":true,"dashboard":true}`), monedas, false,
	)
	mock.ExpectQuery(UpsertSQL).WithArgs(
		"120@g.us", "NUEVO", "SOCIO", "USDT", "10",
		contiene{[]string{`"tasas":true`}},
		contiene{[]string{`"porcentaje"`, `"deposito":2.5`, `"pago":1.5`, `"tipo":"D"`, `"polar_D":"+"`, `"polar_P":"-"`}}, false,
	).WillReturnRows(rows)
	writer := NewWriter(mock)
	perfil, err := writer.Guardar(context.Background(), UpsertParams{
		IDGrupo: "120@g.us", Nombre: "NUEVO", Rol: "SOCIO", MonedaBase: "USDT",
		SaldoInicial: "10", Mostrar: model.MostrarVisible,
		Monedas: model.Monedas{"ars": {Deposito: 2.5, Pago: 1.5}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !perfil.Activo || perfil.Nombre != "NUEVO" || perfil.Monedas["ARS"].Deposito != 2.5 {
		t.Fatalf("perfil = %+v", perfil)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWriterDeleteSinFilas(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectExec(DeleteSQL).WithArgs("NADIE").
		WillReturnResult(pgxmock.NewResult("DELETE", 0))
	err = NewWriter(mock).Delete(context.Background(), "NADIE")
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("err = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWriterPlantilla(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	for _, sql := range []string{ApagarTodosSQL, MemorizarSQL, RestaurarSQL} {
		mock.ExpectExec(sql).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	}
	writer := NewWriter(mock)
	if err := writer.ApagarTodos(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Memorizar(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Restaurar(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
