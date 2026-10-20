package hoo

import (
	"math"
	"strconv"
	"time"
)

// texto reproduce `String(item.clave || 0)` del controlador legado: un valor ausente,
// vacío o cero se publica como la cadena "0", y cualquier otro se pasa tal cual.
//
// La regla del `||` importa: el cero es falsy en JavaScript, así que una tasa de cero
// llega al panel como "0" y no como "0.0".
func texto(item map[string]any, clave string) string {
	switch valor := item[clave].(type) {
	case string:
		if valor != "" {
			return valor
		}
	case float64:
		if valor != 0 {
			return strconv.FormatFloat(valor, 'f', -1, 64)
		}
	}
	return "0"
}

// numero reproduce `Number(item.clave || defecto)`: un valor ausente, vacío o cero cae
// al valor por defecto —así, un `usd` en cero se publica como 1— y un texto que no sea
// un número finito devuelve nil, porque JSON no admite NaN y el original publicaba
// null en su lugar.
func numero(item map[string]any, clave string, defecto float64) *float64 {
	switch valor := item[clave].(type) {
	case float64:
		if valor != 0 {
			return &valor
		}
	case string:
		if valor != "" {
			convertido, err := strconv.ParseFloat(valor, 64)
			if err != nil || math.IsInf(convertido, 0) {
				return nil
			}
			if convertido != 0 {
				return &convertido
			}
		}
	}
	return &defecto
}

// fecha devuelve el sello del origen o el instante actual, como el
// `|| new Date().toISOString()`: el borrador siempre llega con fecha.
func fecha(item map[string]any) string {
	if sello, ok := item["created_at"].(string); ok && sello != "" {
		return sello
	}
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}
