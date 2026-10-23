package calc

import (
	"strings"

	"glauk/internal/comprobantes/domain"
)

// EffectivePolarity resuelve la polaridad del contrato para la operación.
//
// La fuente de verdad es polar_D / polar_P. polaridad queda como respaldo para
// perfiles históricos que todavía no distinguían depósito de pago.
func EffectivePolarity(conf CurrencyConfig, tipoLetra string) string {
	deposito := domain.FirstNonEmpty(conf.PolarD, conf.Polaridad, "+")
	switch tipoLetra {
	case "P":
		if pago := strings.TrimSpace(conf.PolarP); pago != "" {
			return pago
		}
		// La polaridad histórica describía solo el depósito: el pago era su
		// contrapartida. Se conserva esa semántica cuando falta polar_P.
		if deposito == "-" {
			return "+"
		}
		return "-"
	case "D":
		return deposito
	default:
		return "+"
	}
}

// NaturalSign aplica la polaridad del contrato y la naturaleza de operación.
func NaturalSign(cfg SocioConfig, tipoOp string) int {
	if !cfg.Active || tipoOp == "A" {
		return 1
	}
	if EffectivePolarity(cfg.Conf, tipoOp) == "-" {
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

// SideSigns resuelve el signo de cada lado con la polaridad natural del
// contrato, la regla de abono y el ancla de herencia. Es la única fuente de
// verdad tanto para el snapshot persistido como para la proyección live.
//
// El abono depende del rol de la matriz: si el socio 1 es FUNDDA, la matriz
// resta y el socio suma; en cualquier otro caso, al revés.
func SideSigns(cfg1, cfg2 SocioConfig, tipoOp, socio1 string) (int, int) {
	if tipoOp == "A" {
		if strings.EqualFold(strings.TrimSpace(socio1), "FUNDDA") {
			return -1, 1
		}
		return 1, -1
	}
	signo1 := NaturalSign(cfg1, tipoOp)
	signo2 := NaturalSign(cfg2, tipoOp)
	if cfg1.Active && cfg2.Active {
		if cfg1.HereDa && !cfg2.HereDa {
			signo1 = -signo2
		} else if cfg2.HereDa && !cfg1.HereDa {
			signo2 = -signo1
		}
	}
	return signo1, signo2
}

// PolarityOf traduce el signo de un importe congelado a polaridad textual.
// La polaridad persistida debe derivarse del importe, nunca de una regla
// paralela que pueda contradecirlo.
func PolarityOf(value float64) string {
	if value < 0 {
		return "-"
	}
	return "+"
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
