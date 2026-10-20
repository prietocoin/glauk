package repo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"

	"glauk/internal/perfiles/model"
)

var columnasPerfil = []string{
	"id_grupo", "nombre", "rol", "moneda_base", "saldo_inicial", "mostrar", "monedas", "herencia",
}

func loggerRepo(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestReaderListarDecodificaSuperset(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	var buf bytes.Buffer
	monedas := []byte(`{"ars":{"porcentaje":{"deposito":2.5,"pago":1.5},"deposito":9}}`)
	rows := pgxmock.NewRows(columnasPerfil).AddRow(
		"120@g.us", "FUNDABLOCK", "SOCIO", "USDT", "0",
		[]byte(`{"tasas":true,"dashboard":true}`), monedas, false,
	)
	mock.ExpectQuery(ListarSQL).WillReturnRows(rows)
	reader := NewReader(mock, loggerRepo(&buf))
	perfiles, err := reader.Listar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(perfiles) != 1 || perfiles[0].Nombre != "FUNDABLOCK" || !perfiles[0].Activo {
		t.Fatalf("perfiles = %+v", perfiles)
	}
	if perfiles[0].Monedas["ars"].Deposito != 2.5 || perfiles[0].Monedas["ars"].Pago != 1.5 {
		t.Fatalf("monedas = %+v", perfiles[0].Monedas)
	}
	if !strings.Contains(buf.String(), "porcentaje gana") || !strings.Contains(buf.String(), "FUNDABLOCK") {
		t.Fatalf("WARN no emitido: %q", buf.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReaderActivos(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(ActivosSQL).WillReturnRows(
		pgxmock.NewRows([]string{"nombre"}).AddRow("FUNDABLOCK").AddRow("GENERAL"),
	)
	reader := NewReader(mock, nil)
	nombres, err := reader.Activos(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(nombres) != 2 || nombres[0] != "FUNDABLOCK" {
		t.Fatalf("nombres = %v", nombres)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReaderPorNombreSinFilas(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(PorNombreSQL).WithArgs("NADIE").WillReturnRows(pgxmock.NewRows(columnasPerfil))
	reader := NewReader(mock, nil)
	_, err = reader.PorNombre(context.Background(), "NADIE")
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("err = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEncodeMonedasCanonico(t *testing.T) {
	data, err := encodeMonedas(model.Monedas{
		" ars ": {Deposito: 2.5, Pago: 1.5},
	})
	if err != nil {
		t.Fatal(err)
	}
	var salida map[string]map[string]any
	if err := json.Unmarshal(data, &salida); err != nil {
		t.Fatal(err)
	}
	if len(salida) != 1 || salida["ARS"] == nil {
		t.Fatalf("código no normalizado: %s", data)
	}
	if len(salida["ARS"]) != 11 {
		t.Fatalf("superset incompleto: %s", data)
	}
	if salida["ARS"]["def"] != "D" || salida["ARS"]["polaridad"] != "+" {
		t.Fatalf("defaults ausentes: %s", data)
	}
	if salida["ARS"]["deposito"] != 2.5 || salida["ARS"]["pago"] != 1.5 {
		t.Fatalf("valores programáticos perdidos: %s", data)
	}
}
