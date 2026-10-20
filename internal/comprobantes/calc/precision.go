package calc

import "math"

const epsilon = 0.0000001

// TruncateAmount replica la precisión legacy: dos decimales por truncado.
func TruncateAmount(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value == 0 {
		return 0
	}
	sign := 1.0
	if value < 0 {
		sign = -1
	}
	absolute := math.Abs(value)
	rounded := math.Round(absolute*1e8) / 1e8
	return sign * (math.Trunc((rounded+epsilon)*100) / 100)
}

// TruncateCommercialRate replica la regla legacy de tasas comerciales.
func TruncateCommercialRate(value float64) float64 {
	absolute := math.Abs(value)
	if absolute == 0 || math.IsNaN(absolute) || math.IsInf(absolute, 0) {
		return 1
	}
	if absolute > 99.99 {
		return math.Trunc(value)
	}
	return math.Trunc((value+epsilon)*100) / 100
}

// TruncateSnapshotRate aplica la precisión adaptativa del snapshot congelado.
func TruncateSnapshotRate(value float64) float64 {
	absolute := math.Abs(value)
	if absolute == 0 || math.IsNaN(absolute) || math.IsInf(absolute, 0) {
		return 1
	}
	rounded := math.Round(absolute*1e8) / 1e8
	var result float64
	switch {
	case rounded > 99.99:
		result = math.Trunc(rounded)
	case rounded >= 10:
		result = math.Trunc((rounded+epsilon)*100) / 100
	default:
		magnitude := math.Floor(math.Log10(rounded))
		factor := math.Pow(10, 2-magnitude)
		result = math.Trunc((rounded+epsilon)*factor) / factor
	}
	return result
}

// AbsPercent normaliza un porcentaje sin importar su signo de origen.
func AbsPercent(value Decimal) float64 {
	return math.Abs(float64(value))
}
