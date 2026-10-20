package model

import "encoding/json"

// Mostrar controla la visibilidad de un socio en los motores de Glauk.
//
// Tasas es la fuente de verdad de H-02. Dashboard no participa de la actividad
// de Tasas, pero se conserva en el mismo JSONB para no perder la configuración
// del panel. El valor por defecto de un alta nueva es visible en ambos motores.
type Mostrar struct {
	Tasas     bool `json:"tasas"`
	Dashboard bool `json:"dashboard"`
}

// UnmarshalJSON aplica el COALESCE de §1.7.3: una clave ausente vale true, igual
// que (mostrar->>'tasas')::boolean con COALESCE(..., true). Un false explícito
// se respeta y apaga al socio; un objeto presente pero vacío no lo apaga.
func (m *Mostrar) UnmarshalJSON(data []byte) error {
	var entrada struct {
		Tasas     *bool `json:"tasas"`
		Dashboard *bool `json:"dashboard"`
	}
	if err := json.Unmarshal(data, &entrada); err != nil {
		return err
	}
	m.Tasas, m.Dashboard = true, true
	if entrada.Tasas != nil {
		m.Tasas = *entrada.Tasas
	}
	if entrada.Dashboard != nil {
		m.Dashboard = *entrada.Dashboard
	}
	return nil
}

// MostrarVisible es el default de un alta sin configuración explícita.
var MostrarVisible = Mostrar{Tasas: true, Dashboard: true}

// MostrarDe devuelve el valor recibido o el default cuando el puntero es nulo.
// Se usa en el alta: un objeto presente, aunque venga vacío, se respeta tal cual.
func MostrarDe(valor *Mostrar) Mostrar {
	if valor == nil {
		return MostrarVisible
	}
	return *valor
}
