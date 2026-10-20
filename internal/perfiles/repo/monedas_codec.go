package repo

import (
	"encoding/json"
	"strings"

	"glauk/internal/perfiles/model"
)

// encodeMonedas es el único serializador de perfiles_glaukov.monedas. Normaliza
// el código ISO a mayúsculas, completa defaults con model.Canonica y emite el
// superset de once claves. Un código vacío se descarta: no representa una moneda
// real y no debe llegar a un jsonb que consumen tres motores.
//
// El resultado es independiente del orden del mapa porque encoding/json ordena
// las claves, lo que hace estables los fixtures y las comparaciones de auditoría.
func encodeMonedas(monedas model.Monedas) ([]byte, error) {
	salida := make(map[string]json.RawMessage, len(monedas))
	for codigo, moneda := range monedas {
		clave := strings.ToUpper(strings.TrimSpace(codigo))
		if clave == "" {
			continue
		}
		data, err := json.Marshal(moneda.Canonica())
		if err != nil {
			return nil, err
		}
		salida[clave] = data
	}
	return json.Marshal(salida)
}
