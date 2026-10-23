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
	mock.ExpectExec(NormalizarMonedasSQL).
		WithArgs(pgxmock.AnyArg(), "FUNDABLOCK", pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
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

func TestMonedasIncompletas(t *testing.T) {
	casos := []struct {
		nombre string
		raw    string
		quiero bool
	}{
		{"solo tasas", `{"ARS":{"porcentaje":{"deposito":2.5,"pago":1.5}}}`, true},
		{"superset", `{"ARS":{"activo":true,"tipo":"D","polar_D":"+","polar_P":"-"}}`, false},
		{"tipo vacío", `{"ARS":{"activo":true,"tipo":"","polar_D":"+","polar_P":"-"}}`, true},
		{"sin polar_P", `{"ARS":{"activo":true,"tipo":"D","polar_D":"+"}}`, true},
		{"inactiva incompleta", `{"ARS":{"activo":false}}`, false},
		{"activo omitido", `{"ARS":{"porcentaje":{"deposito":1}}}`, true},
		{"vacío", `{}`, false},
		{"nulo", ``, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if got := monedasIncompletas([]byte(caso.raw)); got != caso.quiero {
				t.Fatalf("monedasIncompletas(%s) = %v, quiero %v", caso.raw, got, caso.quiero)
			}
		})
	}
}

func TestReaderListarNoReescribeMonedasCompletas(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	completas := []byte(`{"ARS":{"activo":true,"tipo":"P","polar_D":"-","polar_P":"+"}}`)
	rows := pgxmock.NewRows(columnasPerfil).AddRow(
		"120@g.us", "NELSY", "SOCIO", "USDT", "0",
		[]byte(`{"tasas":true}`), completas, false,
	)
	// Sin ExpectExec: el lector no debe reescribir una fila ya canónica.
	mock.ExpectQuery(ListarSQL).WillReturnRows(rows)
	perfiles, err := NewReader(mock, nil).Listar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	moneda := perfiles[0].Monedas["ARS"]
	if moneda.Tipo != "P" || moneda.PolarD != "-" || moneda.PolarP != "+" {
		t.Fatalf("moneda = %+v", moneda)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReaderPorNombreNormalizaMonedasHistoricas(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	historicas := []byte(`{"COP":{"activo":true,"porcentaje":{"deposito":2.5,"pago":1.5}}}`)
	rows := pgxmock.NewRows(columnasPerfil).AddRow(
		"121@g.us", "NELSY", "SOCIO", "COP", "0",
		[]byte(`{"tasas":true}`), historicas, false,
	)
	mock.ExpectQuery(PorNombreSQL).WithArgs("NELSY").WillReturnRows(rows)
	// El payload persistido debe ser el superset canónico, con el porcentaje
	// original intacto, y el JSONB crudo como guarda contra escrituras
	// concurrentes.
	mock.ExpectExec(NormalizarMonedasSQL).
		WithArgs(
			contiene{[]string{`"tipo":"D"`, `"polar_D":"+"`, `"polar_P":"-"`, `"deposito":2.5`, `"pago":1.5`}},
			"NELSY",
			historicas,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	perfil, err := NewReader(mock, nil).PorNombre(context.Background(), "NELSY")
	if err != nil {
		t.Fatal(err)
	}
	moneda := perfil.Monedas["COP"]
	if moneda.Tipo != "D" || moneda.PolarD != "+" || moneda.PolarP != "-" {
		t.Fatalf("moneda no canonicalizada: %+v", moneda)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReaderPorNombreSanitizaAunqueFalleElUpdate(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	var buf bytes.Buffer
	historicas := []byte(`{"COP":{"activo":true}}`)
	rows := pgxmock.NewRows(columnasPerfil).AddRow(
		"121@g.us", "NELSY", "SOCIO", "COP", "0",
		[]byte(`{"tasas":true}`), historicas, false,
	)
	mock.ExpectQuery(PorNombreSQL).WithArgs("NELSY").WillReturnRows(rows)
	mock.ExpectExec(NormalizarMonedasSQL).
		WithArgs(pgxmock.AnyArg(), "NELSY", pgxmock.AnyArg()).
		WillReturnError(errors.New("db caída"))
	perfil, err := NewReader(mock, loggerRepo(&buf)).PorNombre(context.Background(), "NELSY")
	if err != nil {
		t.Fatalf("la lectura no debe fallar por el saneamiento: %v", err)
	}
	moneda := perfil.Monedas["COP"]
	if moneda.Tipo != "D" || moneda.PolarD != "+" || moneda.PolarP != "-" {
		t.Fatalf("moneda no canonicalizada: %+v", moneda)
	}
	if !strings.Contains(buf.String(), "normalizar monedas") {
		t.Fatalf("WARN no emitido: %q", buf.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
