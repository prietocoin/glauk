package dto

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// OptionalFloat distingue un número omitido de un cero explícito.
//
// Acepta números JSON y cadenas simples con coma de miles, por ejemplo
// "1,500.50". Rechaza NaN, Inf y el formato local completo "1.500,50".
type OptionalFloat struct {
	Value float64
	Set   bool
}

// UnmarshalJSON valida y normaliza el valor recibido por el panel.
func (f *OptionalFloat) UnmarshalJSON(data []byte) error {
	f.Value, f.Set = 0, false
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		return nil
	}

	if strings.HasPrefix(raw, `"`) {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return fmt.Errorf("valor numérico inválido")
		}
		raw = strings.TrimSpace(text)
	}
	if raw == "" || strings.EqualFold(raw, "null") {
		return nil
	}
	if strings.Contains(raw, ",") {
		if strings.Contains(raw, ".") &&
			strings.LastIndex(raw, ",") > strings.LastIndex(raw, ".") {
			return fmt.Errorf("valor numérico inválido")
		}
		raw = strings.ReplaceAll(raw, ",", "")
	}

	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return fmt.Errorf("valor numérico inválido")
	}
	f.Value, f.Set = value, true
	return nil
}

// Or devuelve el valor recibido o el fallback si el campo fue omitido.
func (f OptionalFloat) Or(fallback float64) float64 {
	if f.Set {
		return f.Value
	}
	return fallback
}
