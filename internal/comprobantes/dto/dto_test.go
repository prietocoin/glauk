package dto

import (
	"encoding/json"
	"testing"
)

func TestOptionalFloatAbsentVsZero(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		want    float64
		set     bool
		wantErr bool
	}{
		{name: "omitido", payload: `{}`, set: false},
		{name: "null", payload: `{"value":null}`, set: false},
		{name: "cero_numero", payload: `{"value":0}`, want: 0, set: true},
		{name: "cero_texto", payload: `{"value":"0"}`, want: 0, set: true},
		{name: "decimal", payload: `{"value":12.5}`, want: 12.5, set: true},
		{name: "miles", payload: `{"value":"1,500.50"}`, want: 1500.5, set: true},
		{name: "formato_local", payload: `{"value":"1.500,50"}`, wantErr: true},
		{name: "nan", payload: `{"value":"NaN"}`, wantErr: true},
		{name: "infinito", payload: `{"value":"1e309"}`, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body struct {
				Value OptionalFloat `json:"value"`
			}
			err := json.Unmarshal([]byte(tc.payload), &body)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, esperado error=%v", err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if body.Value.Set != tc.set || body.Value.Value != tc.want {
				t.Fatalf("OptionalFloat = %+v", body.Value)
			}
		})
	}
}

func TestOptionalFloatOrPreservesExplicitZero(t *testing.T) {
	var zero OptionalFloat
	if err := json.Unmarshal([]byte(`0`), &zero); err != nil {
		t.Fatal(err)
	}
	if got := zero.Or(99); got != 0 {
		t.Fatalf("Or devolvió %v para cero explícito", got)
	}
	if got := (OptionalFloat{}).Or(99); got != 99 {
		t.Fatalf("Or devolvió %v para ausente", got)
	}
}

func TestLiquidarInputDefaultsAndExplicitZero(t *testing.T) {
	var empty LiquidarInput
	if err := json.Unmarshal([]byte(`{}`), &empty); err != nil {
		t.Fatal(err)
	}
	liq := empty.ToLiquidacion()
	if liq.Socio1 != DefaultSocio || liq.TipoOp1 != DefaultTipoOp ||
		liq.Tasa1 != 1 || liq.Tasa2 != 1 || liq.LoteTasa != DefaultLoteTasa {
		t.Fatalf("defaults inesperados: %+v", liq)
	}
	if liq.Socio2 == nil || *liq.Socio2 != DefaultSocio {
		t.Fatalf("socio_2 por defecto = %v", liq.Socio2)
	}

	var explicit LiquidarInput
	if err := json.Unmarshal([]byte(`{"tasa_1":0,"monto_1":0,"socio_2":""}`), &explicit); err != nil {
		t.Fatal(err)
	}
	liq = explicit.ToLiquidacion()
	if liq.Tasa1 != 0 || liq.Monto1 != 0 {
		t.Fatalf("cero explícito perdido: %+v", liq)
	}
	if liq.Socio2 == nil || *liq.Socio2 != DefaultSocio {
		t.Fatalf("socio_2 vacío = %v", liq.Socio2)
	}
}

func TestUpdateAliases(t *testing.T) {
	payload := `{"socio1":"SOCIO_A","socio2":"SOCIO_B","moneda_recibo":"COP",
		"id_tasa":"T009","tipo_op":"P","monto_1":10,"m1_socio":25.5}`
	var input UpdateInput
	if err := json.Unmarshal([]byte(payload), &input); err != nil {
		t.Fatal(err)
	}
	if input.Socio1Nombre() != "SOCIO_A" || input.Socio2Nombre() != "SOCIO_B" {
		t.Fatalf("alias de socios no resueltos: %+v", input)
	}
	if input.MonedaValor() != "COP" || input.Lote() != "T009" || input.TipoValor() != "P" {
		t.Fatalf("alias financieros no resueltos: %+v", input)
	}
	if value, ok := input.Monto1Valor(); !ok || value != 10 {
		t.Fatalf("monto_1 ganó sobre m1_socio: %v %v", value, ok)
	}
	queue := input.QueueUpdate()
	if queue.NombreSocio1 == nil || *queue.NombreSocio1 != "SOCIO_A" ||
		queue.Estado == nil || *queue.Estado != "PROCESADO" ||
		queue.Conteo == nil || *queue.Conteo != 1 {
		t.Fatalf("queue defaults inesperados: %+v", queue)
	}
}
