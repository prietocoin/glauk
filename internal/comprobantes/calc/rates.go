package calc

import (
	"encoding/json"
	"math"
)

var defaultRates = map[string]float64{
	"USD":   1.0,
	"USDT":  1.0,
	"PYUSD": 1.2,
	"ECU":   1.0,
	"PAN":   1.0,
}

// DefaultRates devuelve una copia segura de los defaults canónicos.
func DefaultRates() map[string]float64 {
	return MergeRates(nil)
}

// MergeRates superpone el JSONB del lote sobre los defaults históricos.
func MergeRates(values map[string]float64) map[string]float64 {
	merged := make(map[string]float64, len(defaultRates)+len(values))
	for code, value := range defaultRates {
		merged[code] = value
	}
	for code, value := range values {
		merged[CurrencyOf(code)] = value
	}
	return merged
}

// SnapshotRate devuelve una tasa finita y no nula, o 1.
func SnapshotRate(tasas map[string]float64, code string) float64 {
	if value, ok := tasas[code]; ok && value != 0 &&
		!math.IsNaN(value) && !math.IsInf(value, 0) {
		return value
	}
	return 1
}

// DecodeRates acepta el JSONB de tasas o su representación como string JSON.
func DecodeRates(raw []byte) map[string]float64 {
	result := map[string]float64{}
	if len(raw) == 0 {
		return result
	}
	var values map[string]Decimal
	if err := json.Unmarshal(raw, &values); err != nil {
		var encoded string
		if json.Unmarshal(raw, &encoded) != nil {
			return result
		}
		if err := json.Unmarshal([]byte(encoded), &values); err != nil {
			return result
		}
	}
	for code, value := range values {
		result[CurrencyOf(code)] = float64(value)
	}
	return result
}

// RateOr devuelve una tasa comercial positiva o 1.
func RateOr(tasas map[string]float64, code string) float64 {
	if value, ok := tasas[code]; ok && value > 0 {
		return value
	}
	return 1
}

// ValueOr protege apuntadores nulos o NaN.
func ValueOr(value *float64, fallback float64) float64 {
	if value == nil || math.IsNaN(*value) {
		return fallback
	}
	return *value
}

// TasaOrOne garantiza que una tasa congelada nunca sea cero.
func TasaOrOne(value *float64) float64 {
	result := ValueOr(value, 1)
	if result == 0 {
		return 1
	}
	return result
}
