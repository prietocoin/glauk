package repo

import (
	"context"
	"encoding/json"
	"strings"

	"glauk/internal/perfiles/model"
)

// clavesCanonicasMoneda son las claves mínimas que Comprobantes necesita para
// clasificar una operación sin recurrir a heurísticas. Si alguna falta en una
// divisa activa, el motor financiero cae a defaults y puede invertir el signo.
var clavesCanonicasMoneda = [...]string{"tipo", "polar_D", "polar_P"}

// monedasIncompletas informa si alguna divisa activa del JSONB crudo omite
// tipo, polar_D o polar_P. Una clave presente con cadena vacía cuenta como
// ausente: el codec la reemplazaría igual con el default canónico. Las divisas
// inactivas no se tocan para no reescribir configuración que el operador apagó.
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
		for _, clave := range clavesCanonicasMoneda {
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

// normalizarMonedas persiste el superset canónico cuando la fila leída quedó
// incompleta. El perfil ya viene canonicalizado en memoria por el codec, así que
// el socio ve los valores correctos aunque el UPDATE falle o pierda la carrera
// contra otra escritura. El fallo solo se registra: la lectura no se rompe por
// un saneamiento de datos históricos.
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
