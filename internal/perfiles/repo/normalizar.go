package repo

import (
	"context"
	"encoding/json"
	"strings"

	"glauk/internal/perfiles/model"
)

// clavesCanonicasMoneda es el contrato exacto de una divisa en
// perfiles_glaukov.monedas. Cualquier clave adicional es deuda histórica
// (def, dep_op, pago_op, polaridad, deposito, pago) y se elimina al normalizar.
var clavesCanonicasMoneda = map[string]struct{}{
	"activo": {}, "tipo": {}, "polar_D": {}, "polar_P": {}, "porcentaje": {},
}

// monedasIncompletas informa si alguna divisa activa del JSONB crudo no cumple
// el contrato canónico: falta alguna de sus cinco claves, sobra una clave
// legacy o tipo/polar_D/polar_P llegan vacíos. Una cadena vacía cuenta como
// ausente porque el codec la reemplazaría con el default. Las divisas inactivas
// no se tocan para no reescribir configuración que el operador apagó.
func monedasIncompletas(raw []byte) bool {
	if len(raw) == 0 {
		return false
	}
	var monedas map[string]json.RawMessage
	if err := json.Unmarshal(raw, &monedas); err != nil {
		return false
	}
	for _, crudo := range monedas {
		var moneda map[string]json.RawMessage
		if err := json.Unmarshal(crudo, &moneda); err != nil {
			continue
		}
		if !monedaActiva(moneda) {
			continue
		}
		if len(moneda) != len(clavesCanonicasMoneda) {
			return true
		}
		for clave := range moneda {
			if _, ok := clavesCanonicasMoneda[clave]; !ok {
				return true
			}
		}
		for _, clave := range []string{"tipo", "polar_D", "polar_P"} {
			if textoCrudo(moneda[clave]) == "" {
				return true
			}
		}
	}
	return false
}

// monedaActiva replica el default del codec: una divisa sin activo explícito
// está activa. Solo un false literal la excluye del saneamiento.
func monedaActiva(moneda map[string]json.RawMessage) bool {
	raw, ok := moneda["activo"]
	if !ok {
		return true
	}
	var activo bool
	if err := json.Unmarshal(raw, &activo); err != nil {
		return true
	}
	return activo
}

// textoCrudo devuelve el contenido de una clave de texto o vacío si está
// ausente, es null o no es una cadena.
func textoCrudo(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var texto string
	if err := json.Unmarshal(raw, &texto); err != nil {
		return ""
	}
	return strings.TrimSpace(texto)
}

// normalizarMonedas persiste el contrato canónico cuando la fila leída quedó
// incompleta o conserva claves legacy. El perfil ya viene canonicalizado en
// memoria por el codec, así que el socio ve los valores correctos aunque el
// UPDATE falle o pierda la carrera contra otra escritura. El fallo solo se
// registra: la lectura no se rompe por un saneamiento de datos históricos.
func (r *Reader) normalizarMonedas(ctx context.Context, perfil model.Perfil, crudo []byte) {
	if !monedasIncompletas(crudo) {
		return
	}
	canonico, err := encodeMonedas(perfil.Monedas)
	if err != nil {
		r.log.Warn("perfiles: no se pudieron codificar monedas canónicas",
			"socio", perfil.Nombre, "error", err)
		return
	}
	if _, err := r.db.Exec(ctx, NormalizarMonedasSQL, canonico, perfil.Nombre, crudo); err != nil {
		r.log.Warn("perfiles: no se pudieron normalizar monedas",
			"socio", perfil.Nombre, "error", err)
	}
}
