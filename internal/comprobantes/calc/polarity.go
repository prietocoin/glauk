package calc

import (
	"strings"

	"glauk/internal/comprobantes/domain"
)

// NaturalSign aplica la polaridad del contrato y la naturaleza de operación.
func NaturalSign(cfg SocioConfig, tipoOp string) int {
	if !cfg.Active || tipoOp == "A" {
		return 1
	}
	depositoSuma := cfg.Conf.Polaridad == "+"
	if tipoOp == "D" {
		if depositoSuma {
			return 1
		}
		return -1
	}
	if depositoSuma {
		return -1
	}
	return 1
}

// OperationNature reduce el tipo manual a su letra canónica.
func OperationNature(value string) string {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "D"
	}
	return normalized[:1]
}

// ResolvePolarity elige polar_D, polar_P o la regla especial de abono.
func ResolvePolarity(conf CurrencyConfig, tipoLetra string, esSocio1 bool) string {
	switch tipoLetra {
	case "A":
		if esSocio1 {
			return "+"
		}
		return "-"
	case "P":
		return domain.FirstNonEmpty(conf.PolarP, "-")
	default:
		return domain.FirstNonEmpty(conf.PolarD, "+")
	}
}

// SignFor traduce la polaridad textual al signo numérico.
func SignFor(polaridad string) int {
	if strings.TrimSpace(polaridad) == "-" {
		return -1
	}
	return 1
}

// IsGeneral identifica la entidad matriz o un nombre vacío.
func IsGeneral(name string) bool {
	return strings.EqualFold(strings.TrimSpace(name), "GENERAL") ||
		strings.TrimSpace(name) == ""
}

// OptionalSocio devuelve nil para GENERAL y el nombre limpio para el resto.
func OptionalSocio(name string) *string {
	trimmed := strings.TrimSpace(name)
	if IsGeneral(trimmed) {
		return nil
	}
	return &trimmed
}

// CurrencyOf normaliza un código de divisa con fallback USDT.
func CurrencyOf(value string) string {
	return domain.NormalizeCurrency(value, "USDT")
}
