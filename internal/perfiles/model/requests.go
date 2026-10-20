package model

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// FlexibleFloat acepta un número JSON o una cadena numérica, como enviaba Alpine.
// Conserva el texto original para que saldo_inicial mantenga la precisión de node-pg.
type FlexibleFloat string

// UnmarshalJSON normaliza null y vacío a "0" y rechaza valores no finitos.
func (f *FlexibleFloat) UnmarshalJSON(raw []byte) error {
	texto := strings.TrimSpace(string(raw))
	if texto == "" || texto == "null" {
		*f = "0"
		return nil
	}
	if len(texto) >= 2 && texto[0] == '"' {
		if err := json.Unmarshal(raw, &texto); err != nil {
			return fmt.Errorf("número inválido: %w", err)
		}
	}
	texto = strings.TrimSpace(texto)
	if texto == "" {
		*f = "0"
		return nil
	}
	valor, err := strconv.ParseFloat(texto, 64)
	if err != nil || math.IsNaN(valor) || math.IsInf(valor, 0) {
		return fmt.Errorf("número inválido: %q", texto)
	}
	*f = FlexibleFloat(texto)
	return nil
}

// Float64 devuelve el valor numérico; un FlexibleFloat vacío vale cero.
func (f FlexibleFloat) Float64() float64 {
	valor, _ := strconv.ParseFloat(string(f), 64)
	return valor
}

// String conserva el contrato de saldo_inicial como cadena.
func (f FlexibleFloat) String() string {
	if strings.TrimSpace(string(f)) == "" {
		return "0"
	}
	return string(f)
}
