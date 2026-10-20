package domain

import "strings"

// NormalizeCurrency devuelve el código ISO en mayúsculas o el fallback.
func NormalizeCurrency(value, fallback string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

// FirstNonEmpty devuelve el primer valor con contenido después de limpiarlo.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// PositiveOrOne protege divisores financieros sin alterar valores válidos.
func PositiveOrOne(value float64) float64 {
	if value > 0 {
		return value
	}
	return 1
}

// Money conserva un monto junto con su divisa de origen.
type Money struct {
	Amount   float64
	Currency string
}

// NewMoney normaliza la divisa y conserva el monto recibido.
func NewMoney(amount float64, currency string) Money {
	return Money{Amount: amount, Currency: NormalizeCurrency(currency, "USDT")}
}
