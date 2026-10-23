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

// SideSigns resuelve el signo de cada lado con la polaridad natural del
// contrato, la regla de abono y el ancla de herencia. Es la única fuente de
// verdad tanto para el snapshot congelado como para la proyección live.
//
// Si exactamente un lado hereda de FUNDDA, su signo se invierte para quedar
// opuesto al lado no heredante. Si ambos heredan, no hay override.
func SideSigns(cfg1, cfg2 SocioConfig, tipoOp string) (int, int) {
	signo1 := NaturalSign(cfg1, tipoOp)
	signo2 := NaturalSign(cfg2, tipoOp)
	if tipoOp == "A" {
		// El abono es contrapartida: socio 1 suma, socio 2 resta.
		signo1, signo2 = 1, -1
	}
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
