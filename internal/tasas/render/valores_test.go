package render

import (
	"math"
	"testing"
)

// TestTasaConvierteLoNoFinitoEnNil protege el contrato JSON: NaN e infinito no
// existen en JSON y el servicio original los publicaba como null.
func TestTasaConvierteLoNoFinitoEnNil(t *testing.T) {
	casos := []struct {
		nombre  string
		entrada float64
		nulo    bool
	}{
		{"finita", 1250, false},
		{"cero", 0, false},
		{"NaN", math.NaN(), true},
		{"infinito", math.Inf(1), true},
		{"infinito negativo", math.Inf(-1), true},
	}
	for _, caso := range casos {
		obtenido := tasa(caso.entrada)
		if (obtenido == nil) != caso.nulo {
			t.Errorf("%s: nil = %v, se esperaba %v", caso.nombre, obtenido == nil, caso.nulo)
			continue
		}
		if obtenido != nil && *obtenido != caso.entrada {
			t.Errorf("%s: valor = %v, se esperaba %v", caso.nombre, *obtenido, caso.entrada)
		}
	}
}

// TestTextoDeTasaImitaAJavaScript fija el formato del valor pintado: sin ceros
// sobrantes, sin notación científica y con "null" para lo no finito.
func TestTextoDeTasaImitaAJavaScript(t *testing.T) {
	cero, entero, decimal := 0.0, 1250.0, 17.8
	casos := []struct {
		nombre string
		valor  *float64
		texto  string
	}{
		{"nulo", nil, "null"},
		{"cero", &cero, "0"},
		{"entero", &entero, "1250"},
		{"decimal", &decimal, "17.8"},
	}
	for _, caso := range casos {
		if obtenido := textoDeTasa(caso.valor); obtenido != caso.texto {
			t.Errorf("%s: texto = %q, se esperaba %q", caso.nombre, obtenido, caso.texto)
		}
	}
}

// TestOrElseRespetaElOrden cubre la regla del `||` del servicio original: gana el
// primer valor no vacío y, si todos lo están, queda la cadena vacía.
func TestOrElseRespetaElOrden(t *testing.T) {
	if obtenido := orElse("", "Argentina", "ARS"); obtenido != "Argentina" {
		t.Errorf("orElse = %q, se esperaba Argentina", obtenido)
	}
	if obtenido := orElse("", ""); obtenido != "" {
		t.Errorf("orElse = %q, se esperaba la cadena vacía", obtenido)
	}
}
