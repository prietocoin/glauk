package calc

import "strings"

const (
	// socioAbonoCondicionado es el socio alcanzado por la regla especial.
	socioAbonoCondicionado = "IRIS"
	// divisaAbonoCondicionada es la única divisa alcanzada por la regla.
	divisaAbonoCondicionada = "COP"
)

// NatureForAbono aplica la regla de negocio de IRIS en COP.
//
// El motor históricamente convertía en abono ("A") toda operación cuya divisa
// coincidiera con la moneda base del socio 1. Para IRIS en COP eso generaba
// abonos implícitos que no existen: solo es abono si el caption del comprobante
// lo declara de forma explícita. Sin esa indicación se usa el tipo contractual
// de la divisa (por ejemplo "D") y, por tanto, sus polar_D / polar_P.
//
// Fuera de IRIS+COP la naturaleza se devuelve intacta: la regla no altera el
// comportamiento de los demás socios ni divisas.
func NatureForAbono(tipo, socio1, socio2, divisa, caption string, conf CurrencyConfig) string {
	naturaleza := OperationNature(tipo)
	if naturaleza != "A" {
		return naturaleza
	}
	if !esAbonoCondicionado(socio1, socio2, divisa) {
		return "A"
	}
	if contieneIndicadorAbono(caption) {
		return "A"
	}
	return tipoContractual(conf)
}

// esAbonoCondicionado indica si la operación involucra a IRIS en COP, sin
// importar si aparece como socio 1 o socio 2.
func esAbonoCondicionado(socio1, socio2, divisa string) bool {
	if !strings.EqualFold(strings.TrimSpace(divisa), divisaAbonoCondicionada) {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(socio1), socioAbonoCondicionado) ||
		strings.EqualFold(strings.TrimSpace(socio2), socioAbonoCondicionado)
}

// contieneIndicadorAbono verifica, sin distinguir mayúsculas, si el caption del
// comprobante declara un abono. Un caption vacío o sin estas palabras clave no
// califica: la ausencia de evidencia no se interpreta como abono.
func contieneIndicadorAbono(caption string) bool {
	texto := strings.ToLower(strings.TrimSpace(caption))
	if texto == "" {
		return false
	}
	for _, indicador := range []string{"abono", "abona", "abonar"} {
		if strings.Contains(texto, indicador) {
			return true
		}
	}
	return false
}

// tipoContractual lee la naturaleza configurada en perfiles_glaukov. Solo D y P
// son contratos operativos; cualquier valor ausente o desconocido cae en D, que
// es el default histórico del motor.
func tipoContractual(conf CurrencyConfig) string {
	switch tipo := OperationNature(conf.Tipo); tipo {
	case "D", "P":
		return tipo
	default:
		return "D"
	}
}
