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

func TestMonedaRoundTripSuperset(t *testing.T) {
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
		if len(claves) != 11 {
			t.Fatalf("superset tiene %d claves: %s", len(claves), primera)
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
