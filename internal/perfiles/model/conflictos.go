package model

import (
	"sort"
	"strings"
)

// Conflicto documenta un valor legacy descartado en favor de porcentaje.
//
// El modelo detecta el conflicto durante UnmarshalJSON, pero no conoce el nombre
// del socio ni emite logs. Las capas de servicio y repositorio aplanan esta lista
// y emiten un WARN por cada entrada, con el nombre del socio y el código ISO.
type Conflicto struct {
	Moneda     string  `json:"moneda"`
	Campo      string  `json:"campo"`
	Descartado float64 `json:"descartado"`
	Conservado float64 `json:"conservado"`
}

// Conflictos devuelve una copia de los valores legacy que fueron descartados.
func (m Moneda) Conflictos() []Conflicto {
	return append([]Conflicto(nil), m.conflictos...)
}

// Conflictos aplana los conflictos del mapa y fija el código de cada moneda.
// Ordena el resultado para que los tests y los reportes de auditoría sean
// deterministas aunque el mapa se recorra en orden aleatorio.
func (m Monedas) Conflictos() []Conflicto {
	conflictos := make([]Conflicto, 0)
	for codigo, moneda := range m {
		for _, conflicto := range moneda.Conflictos() {
			conflicto.Moneda = strings.ToUpper(strings.TrimSpace(codigo))
			conflictos = append(conflictos, conflicto)
		}
	}
	sort.Slice(conflictos, func(i, j int) bool {
		if conflictos[i].Moneda == conflictos[j].Moneda {
			return conflictos[i].Campo < conflictos[j].Campo
		}
		return conflictos[i].Moneda < conflictos[j].Moneda
	})
	return conflictos
}
