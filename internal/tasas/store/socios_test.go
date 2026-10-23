package store

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"

	perfilrepo "glauk/internal/perfiles/repo"
	"glauk/internal/tasas/domain"
)

// monedasContiene verifica que el JSONB codificado conserve las claves que Tasas
// no interpreta pero debe transportar.
type monedasContiene struct{ claves []string }

func (m monedasContiene) Match(valor any) bool {
	raw, ok := valor.([]byte)
	if !ok {
		return false
	}
	for _, clave := range m.claves {
		if !bytes.Contains(raw, []byte(clave)) {
			return false
		}
	}
	return true
}

// columnasHeredadas es la tabla tal como está en producción: no tiene `activo`.
var columnasHeredadas = []string{"nombre", "id_grupo", "moneda_base", "monedas", "modo_prueba", "rol"}

// sqlSocioPorNombre es la lectura literal: si alguien vuelve a enumerar columnas en
// el SELECT, esta prueba deja de coincidir y avisa.
const sqlSocioPorNombre = `SELECT * FROM perfiles_glaukov WHERE LOWER(nombre) = LOWER($1)`

// TestSocioSinColumnaActivo es la regresión del incidente: la lectura pedía `activo`,
// PostgreSQL respondía 42703 y /socios caía entero.
func TestSocioSinColumnaActivo(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(sqlSocioPorNombre).WithArgs("Ana").WillReturnRows(pgxmock.NewRows(columnasHeredadas).
		AddRow("Ana", nil, "USDT", []byte(`{"COP":{"activo":true}}`), nil, ptr("ADMIN")))

	socio, err := SocioByName(context.Background(), mock, "Ana")
	if err != nil {
		t.Fatalf("la tabla sin activo debe leerse igual: %v", err)
	}
	if socio.Nombre != "Ana" || socio.MonedaBase != "USDT" || socio.Rol != "ADMIN" {
		t.Fatalf("socio mal traducido: %+v", socio)
	}
	if socio.Activo != nil || socio.ModoPrueba != nil {
		t.Fatalf("una columna ausente debe quedar nula, no falsa: %+v", socio)
	}
	if moneda := socio.Monedas["COP"]; moneda == nil || moneda.Activo == nil || !*moneda.Activo {
		t.Fatalf("la configuración de monedas no se interpretó: %+v", socio.Monedas)
	}
}

// TestSocioConColumnasMinimas cubre una tabla aún más pobre: sin grupo, sin base y
// sin rol. Debe leerse con valores cero en vez de fallar.
func TestSocioConColumnasMinimas(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(sqlSocioPorNombre).WithArgs("Ana").
		WillReturnRows(pgxmock.NewRows([]string{"nombre", "monedas"}).AddRow("Ana", []byte(`{}`)))

	socio, err := SocioByName(context.Background(), mock, "Ana")
	if err != nil || socio.Nombre != "Ana" {
		t.Fatalf("una tabla con solo nombre y monedas debe leerse: %+v, %v", socio, err)
	}
	if socio.IDGrupo != "" || socio.MonedaBase != "" || socio.Rol != "" {
		t.Fatalf("las columnas ausentes deben quedar vacías: %+v", socio)
	}
}

// TestSocioInexistente conserva el contrato de "no había filas": ErrNotFound, que la
// capa HTTP traduce a 404.
// TestSocioLeeMostrar fija la fuente de verdad de H-02: el JSONB mostrar.tasas
// debe viajar desde la fila hasta el socio que decide la publicación.
func TestSocioLeeMostrar(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(sqlSocioPorNombre).WithArgs("Ana").
		WillReturnRows(pgxmock.NewRows([]string{"nombre", "monedas", "mostrar"}).
			AddRow("Ana", []byte(`{}`), []byte(`{"tasas":true}`)))
	socio, err := SocioByName(context.Background(), mock, "Ana")
	if err != nil || socio.Mostrar.Tasas == nil || !*socio.Mostrar.Tasas {
		t.Fatalf("mostrar.tasas no se interpretó: %+v, %v", socio.Mostrar, err)
	}
}

func TestSocioInexistente(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(sqlSocioPorNombre).WithArgs("Nadie").WillReturnRows(pgxmock.NewRows(columnasHeredadas))
	if _, err := SocioByName(context.Background(), mock, "Nadie"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("se esperaba ErrNotFound, llegó: %v", err)
	}
}

// TestUpdateSocioEscribeSoloLasColumnasQueExisten fija la sentencia que se arma:
// `activo` queda fuera porque la tabla no la tiene, y los interruptores se escriben
// con COALESCE para que un nulo signifique "no lo toques".
func TestUpdateSocioEscribeSoloLasColumnasQueExisten(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(`SELECT * FROM perfiles_glaukov LIMIT 0`).WillReturnRows(pgxmock.NewRows(columnasHeredadas))
	config := []byte(`{"COP":{"activo":true,"porcentaje":{"deposito":-2,"pago":3}}}`)
	sentencia, args, err := perfilrepo.SentenciaSocioTasas(
		map[string]bool{"monedas": true, "modo_prueba": true, "rol": true},
		perfilrepo.ActualizacionSocioTasas{Monedas: config}, "Ana",
	)
	if err != nil {
		t.Fatal(err)
	}
	args[0] = pgxmock.AnyArg()
	mock.ExpectQuery(sentencia).WithArgs(args...).
		WillReturnRows(pgxmock.NewRows(columnasHeredadas).AddRow("Ana", nil, nil, config, nil, ptr("ADMIN")))

	socio, err := UpdateSocio(context.Background(), mock, "Ana", SocioUpdate{
		Monedas: map[string]*domain.Moneda{"COP": {Activo: ptr(true), Porcentaje: domain.Porcentajes{Deposito: -2, Pago: 3}}},
	})
	if err != nil || socio.Rol != "ADMIN" || socio.Monedas["COP"].Porcentaje.Pago != 3 {
		t.Fatalf("UpdateSocio devolvió %+v, %v", socio, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("quedaron expectativas sin cumplir: %v", err)
	}
}

// TestUpdateSocioConservaPolaridades es la regresión del modal de Perfiles: el
// panel de Tasas reenvía el mapa completo de monedas y el UPDATE no puede
// reducirlo a activo/porcentaje.
func TestUpdateSocioConservaPolaridades(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(`SELECT * FROM perfiles_glaukov LIMIT 0`).WillReturnRows(pgxmock.NewRows(columnasHeredadas))
	config := []byte(`{"COP":{"activo":true,"porcentaje":{"deposito":2.5,"pago":3},"def":"D","deposito":2.5,"dep_op":"-","pago":3,"pago_op":"+","tipo":"D","polar_D":"-","polar_P":"+","polaridad":"+"}}`)
	sentencia, args, err := perfilrepo.SentenciaSocioTasas(
		map[string]bool{"monedas": true, "modo_prueba": true, "rol": true},
		perfilrepo.ActualizacionSocioTasas{Monedas: config}, "Ana",
	)
	if err != nil {
		t.Fatal(err)
	}
	args[0] = monedasContiene{[]string{`"tipo":"D"`, `"polar_D":"-"`, `"polar_P":"+"`, `"porcentaje"`}}
	mock.ExpectQuery(sentencia).WithArgs(args...).
		WillReturnRows(pgxmock.NewRows(columnasHeredadas).AddRow("Ana", nil, nil, config, nil, ptr("ADMIN")))

	socio, err := UpdateSocio(context.Background(), mock, "Ana", SocioUpdate{
		Monedas: map[string]*domain.Moneda{"COP": {
			Activo:     ptr(true),
			Porcentaje: domain.Porcentajes{Deposito: 2.5, Pago: 3},
			Tipo:       "D",
			PolarD:     "-",
			PolarP:     "+",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	moneda := socio.Monedas["COP"]
	if moneda == nil || moneda.Tipo != "D" || moneda.PolarD != "-" || moneda.PolarP != "+" {
		t.Fatalf("la lectura perdió las polaridades: %+v", moneda)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("quedaron expectativas sin cumplir: %v", err)
	}
}
