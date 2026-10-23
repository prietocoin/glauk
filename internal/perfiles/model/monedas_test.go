package model

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestMonedaTresFormas(t *testing.T) {
	casos := []struct {
		nombre   string
		raw      string
		deposito float64
		pago     float64
		activo   bool
	}{
		{"tasas", `{"porcentaje":{"deposito":2.5,"pago":1.5}}`, 2.5, 1.5, true},
		{"perfiles", `{"def":"D","deposito":2.5,"dep_op":"+","pago":1.5,"pago_op":"-"}`, 2.5, 1.5, true},
		{"comprobantes", `{"tipo":"D","polar_D":"+","polar_P":"-","polaridad":"+","porcentaje":{"deposito":3,"pago":2}}`, 3, 2, true},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			var moneda Moneda
			if err := json.Unmarshal([]byte(caso.raw), &moneda); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if moneda.Deposito != caso.deposito || moneda.Pago != caso.pago {
				t.Fatalf("valores = %v/%v, quiero %v/%v", moneda.Deposito, moneda.Pago, caso.deposito, caso.pago)
			}
			if moneda.Porcentaje.Deposito != caso.deposito || moneda.Porcentaje.Pago != caso.pago {
				t.Fatalf("porcentaje = %+v", moneda.Porcentaje)
			}
			if moneda.Activo != caso.activo || moneda.Def != "D" || moneda.Tipo != "D" {
				t.Fatalf("defaults = %+v", moneda)
			}
			if moneda.DepOp != "+" || moneda.PagoOp != "-" || moneda.PolarD != "+" || moneda.PolarP != "-" || moneda.Polaridad != "+" {
				t.Fatalf("operadores = %+v", moneda)
			}
		})
	}
}

func TestMonedaMapeaContratoPolaridadYActivo(t *testing.T) {
	raw := `{"activo":false,"tipo":"P","polar_D":"-","polar_P":"+",` +
		`"polaridad":"-","porcentaje":{"deposito":"2.5","pago":4}}`
	var moneda Moneda
	if err := json.Unmarshal([]byte(raw), &moneda); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if moneda.Activo {
		t.Fatalf("activo = true, quiero false")
	}
	if moneda.PolarD != "-" || moneda.PolarP != "+" || moneda.Polaridad != "-" {
		t.Fatalf("polaridad = %+v", moneda)
	}
	if moneda.Porcentaje.Deposito != 2.5 || moneda.Porcentaje.Pago != 4 ||
		moneda.Deposito != 2.5 || moneda.Pago != 4 {
		t.Fatalf("porcentaje = %+v", moneda)
	}
}

func TestMonedaRoundTripCanonico(t *testing.T) {
	canonicas := map[string]struct{}{
		"activo": {}, "tipo": {}, "polar_D": {}, "polar_P": {}, "porcentaje": {},
	}
	fixtures := []string{
		`{"porcentaje":{"deposito":2.5,"pago":1.5}}`,
		`{"def":"D","deposito":2.5,"dep_op":"+","pago":1.5,"pago_op":"-"}`,
		`{"tipo":"D","polar_D":"+","polar_P":"-","polaridad":"+","porcentaje":{"deposito":3,"pago":2}}`,
	}
	for _, fixture := range fixtures {
		var original Moneda
		if err := json.Unmarshal([]byte(fixture), &original); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		primera, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		var claves map[string]json.RawMessage
		if err := json.Unmarshal(primera, &claves); err != nil {
			t.Fatalf("claves: %v", err)
		}
		if len(claves) != len(canonicas) {
			t.Fatalf("el contrato tiene %d claves, quiero %d: %s", len(claves), len(canonicas), primera)
		}
		for clave := range claves {
			if _, ok := canonicas[clave]; !ok {
				t.Fatalf("clave legacy %q en %s", clave, primera)
			}
		}
		var releida Moneda
		if err := json.Unmarshal(primera, &releida); err != nil {
			t.Fatalf("relectura: %v", err)
		}
		segunda, err := json.Marshal(releida)
		if err != nil {
			t.Fatalf("segundo Marshal: %v", err)
		}
		if !bytes.Equal(primera, segunda) {
			t.Fatalf("round-trip inestable:\n%s\n%s", primera, segunda)
		}
	}
}

func TestPorcentajeGanaSobreLegacy(t *testing.T) {
	raw := `{"porcentaje":{"deposito":5,"pago":2},"deposito":3,"pago":4}`
	var moneda Moneda
	if err := json.Unmarshal([]byte(raw), &moneda); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if moneda.Deposito != 5 || moneda.Pago != 2 {
		t.Fatalf("ganó legacy: %+v", moneda)
	}
	conflictos := moneda.Conflictos()
	if len(conflictos) != 2 {
		t.Fatalf("conflictos = %+v", conflictos)
	}
	if conflictos[0].Campo != "deposito" || conflictos[0].Descartado != 3 || conflictos[0].Conservado != 5 {
		t.Fatalf("conflicto depósito = %+v", conflictos[0])
	}
	if conflictos[1].Campo != "pago" || conflictos[1].Descartado != 4 || conflictos[1].Conservado != 2 {
		t.Fatalf("conflicto pago = %+v", conflictos[1])
	}
}

// TestGuardarRequestConservaPolaridades cubre el payload exacto del modal de
// Perfiles: tipo, polar_D y polar_P deben sobrevivir al Unmarshal y al contrato
// canónico, sin reinyectar los alias legacy.
func TestGuardarRequestConservaPolaridades(t *testing.T) {
	payload := `{"nombre":"X","monedas":{"COP":{"activo":true,"tipo":"P","polar_D":"-","polar_P":"+","porcentaje":{"deposito":2.5,"pago":1.5}}}}`
	var req GuardarRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	moneda := req.Monedas["COP"].Canonica()
	if !moneda.Activo || moneda.Tipo != "P" || moneda.PolarD != "-" || moneda.PolarP != "+" {
		t.Fatalf("el payload perdió su configuración: %+v", moneda)
	}
	if moneda.Porcentaje.Deposito != 2.5 || moneda.Porcentaje.Pago != 1.5 {
		t.Fatalf("porcentaje = %+v", moneda.Porcentaje)
	}
	raw, err := json.Marshal(moneda)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var claves map[string]json.RawMessage
	if err := json.Unmarshal(raw, &claves); err != nil {
		t.Fatalf("claves: %v", err)
	}
	for _, clave := range []string{"activo", "tipo", "polar_D", "polar_P", "porcentaje"} {
		if _, ok := claves[clave]; !ok {
			t.Fatalf("falta %q en %s", clave, raw)
		}
	}
	for _, clave := range []string{"def", "deposito", "dep_op", "pago", "pago_op", "polaridad"} {
		if _, ok := claves[clave]; ok {
			t.Fatalf("clave legacy %q reinyectada en %s", clave, raw)
		}
	}
}

func TestCanonicaConservaPrioridad(t *testing.T) {
	var conflicto Moneda
	if err := json.Unmarshal([]byte(`{"porcentaje":{"deposito":5,"pago":2},"deposito":3,"pago":4}`), &conflicto); err != nil {
		t.Fatal(err)
	}
	if canonica := conflicto.Canonica(); canonica.Deposito != 5 || canonica.Pago != 2 {
		t.Fatalf("Canonica perdió porcentaje: %+v", canonica)
	}
	programatica := Moneda{Deposito: 7, Pago: 3}.Canonica()
	if programatica.Porcentaje.Deposito != 7 || programatica.Pago != 3 || programatica.Def != "D" {
		t.Fatalf("Canonica perdió valores planos: %+v", programatica)
	}
}

// TestMonedaMigraPolaridadLegacy cubre las filas históricas que solo traían una
// polaridad. Esa polaridad describía el depósito; el pago es su contrapartida.
func TestMonedaMigraPolaridadLegacy(t *testing.T) {
	var moneda Moneda
	if err := json.Unmarshal([]byte(`{"polaridad":"-","porcentaje":{"deposito":1,"pago":2}}`), &moneda); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if moneda.PolarD != "-" || moneda.PolarP != "+" || moneda.Polaridad != "-" || moneda.Tipo != "D" {
		t.Fatalf("migración legacy = %+v", moneda)
	}

	var explicita Moneda
	if err := json.Unmarshal([]byte(`{"polar_D":"+","polaridad":"-"}`), &explicita); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if explicita.PolarD != "+" || explicita.PolarP != "-" || explicita.Polaridad != "+" {
		t.Fatalf("polar_D debe mandar sobre polaridad: %+v", explicita)
	}
}

func TestMonedasConflictosOrdenados(t *testing.T) {
	var monedas Monedas
	raw := `{"usd":{"porcentaje":{"deposito":1},"deposito":9},"ars":{"porcentaje":{"pago":2},"pago":8}}`
	if err := json.Unmarshal([]byte(raw), &monedas); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	conflictos := monedas.Conflictos()
	if len(conflictos) != 2 || conflictos[0].Moneda != "ARS" || conflictos[1].Moneda != "USD" {
		t.Fatalf("conflictos = %+v", conflictos)
	}
}
