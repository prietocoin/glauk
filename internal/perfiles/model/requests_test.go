package model

import (
	"encoding/json"
	"testing"
)

func TestFlexibleFloatTolerante(t *testing.T) {
	casos := []struct {
		raw    string
		quiero string
		error  bool
	}{
		{`12.5`, "12.5", false},
		{`"12.5"`, "12.5", false},
		{`null`, "0", false},
		{`""`, "0", false},
		{`"NaN"`, "", true},
		{`"abc"`, "", true},
	}
	for _, caso := range casos {
		var valor FlexibleFloat
		err := json.Unmarshal([]byte(caso.raw), &valor)
		if caso.error {
			if err == nil {
				t.Fatalf("%s: esperaba error", caso.raw)
			}
			continue
		}
		if err != nil || valor.String() != caso.quiero {
			t.Fatalf("%s: valor=%q err=%v", caso.raw, valor, err)
		}
	}
}

func TestMostrarDefaultsH02(t *testing.T) {
	casos := []struct {
		raw              string
		tasas, dashboard bool
	}{
		{`{}`, true, true},
		{`{"tasas":false}`, false, true},
		{`{"dashboard":false}`, true, false},
		{`{"tasas":true,"dashboard":false}`, true, false},
	}
	for _, caso := range casos {
		var mostrar Mostrar
		if err := json.Unmarshal([]byte(caso.raw), &mostrar); err != nil {
			t.Fatalf("%s: %v", caso.raw, err)
		}
		if mostrar.Tasas != caso.tasas || mostrar.Dashboard != caso.dashboard {
			t.Fatalf("%s: %+v", caso.raw, mostrar)
		}
	}
}

func TestGuardarRequestAlias(t *testing.T) {
	raw := `{"socio":" nuevo ","jid":"120@g.us","moneda":"usdt","saldo":"12.5","activo":false}`
	var req GuardarRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if req.Nombre != " nuevo " || req.IDGrupo != "120@g.us" || req.MonedaBase != "usdt" || req.SaldoInicial.String() != "12.5" {
		t.Fatalf("alias no traducidos: %+v", req)
	}
	if req.Mostrar != nil {
		t.Fatalf("mostrar no puede inventarse: %+v", req.Mostrar)
	}
}

func TestGuardarRequestCanonicoGana(t *testing.T) {
	raw := `{"nombre":"CANONICO","socio":"ALIAS","moneda_base":"EUR","moneda":"USD","saldo_inicial":7,"saldo":9}`
	var req GuardarRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if req.Nombre != "CANONICO" || req.MonedaBase != "EUR" || req.SaldoInicial.String() != "7" {
		t.Fatalf("el alias pisó el campo canónico: %+v", req)
	}
}
