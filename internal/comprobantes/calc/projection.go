package calc

import (
	"math"
	"strings"

	"glauk/internal/comprobantes/domain"
)

// ProjectRow transforma una fila cruda en el contrato de auditoría.
//
// Si la fila está liquidada, los importes, tasas, equivalencias y etiquetas de
// operación se toman del snapshot congelado en comprobantes_liq. La proyección
// viva queda únicamente como respaldo para filas no liquidadas o para lados
// que no conservan un snapshot completo.
func ProjectRow(row SourceRow, fundda Profile) domain.Comprobante {
	monto := math.Abs(TruncateAmount(row.Monto))
	liquidado := row.EstaLiquidado
	socio1 := resolveSocio1(row, liquidado)
	socio2 := resolveSocio2(row, liquidado)
	socio2Nombre := ""
	if socio2 != nil {
		socio2Nombre = *socio2
	}
	divisa := domain.NormalizeCurrency(row.Moneda, "COP")
	monedaSocio1 := domain.NormalizeCurrency(row.MonedaSocio1, "USDT")
	monedaSocio2 := domain.NormalizeCurrency(row.MonedaSocio2, "USDT")

	lado1 := frozenSide(row.Tasa1, row.Monto1, row.ME1)
	lado2 := frozenSide(row.Tasa2, row.Monto2, row.ME2)
	tipo1, tipo2 := "", ""
	if liquidado {
		tipo1 = frozenType(row.TipoOp1, nil)
		tipo2 = frozenType(row.TipoOp2, row.TipoOp1)
	}

	var tasa1, monto1, me1, tasa2, monto2, me2 float64
	if liquidado && lado1.ok {
		tasa1, monto1, me1 = lado1.tasa, lado1.monto, lado1.me
	}
	if socio2 == nil {
		// Sin segundo socio no existe lado 2, aunque una fila histórica
		// conserve columnas sueltas.
		tasa2, monto2, me2 = 1, 0, 0
		lado2.ok = false
	} else if liquidado && lado2.ok {
		tasa2, monto2, me2 = lado2.tasa, lado2.monto, lado2.me
	}

	necesitaVivo := !liquidado || !lado1.ok ||
		(socio2 != nil && !lado2.ok) ||
		tipo1 == "" || (socio2 != nil && tipo2 == "")
	if necesitaVivo {
		profile1 := DecodeProfile(row.Socio1Row)
		profile2 := DecodeProfile(row.Socio2Row)
		cfg1 := ResolveLiveSocioConfig(socio1, profile1, divisa, fundda, monedaSocio1)
		cfg2 := ResolveLiveSocioConfig(socio2Nombre, profile2, divisa, fundda, monedaSocio2)
		tipoLetra := operationLetter(row, cfg1.Conf, socio1, socio2Nombre, divisa)
		// Si el snapshot trae la naturaleza pero le falta algún importe, el
		// respaldo vivo debe usar esa misma letra y no la del perfil actual.
		if liquidado {
			if letra := savedLetter(&tipo1); letra != "" {
				tipoLetra = letra
			} else if letra := savedLetter(&tipo2); letra != "" {
				tipoLetra = letra
			}
		}
		tipoTag := tipoLetra + "-" + divisa
		signo1, signo2 := SideSigns(cfg1, cfg2, tipoLetra, socio1)
		liveTasa1, liveMonto1, liveME1, liveTasa2, liveMonto2, liveME2 := CalculateAmounts(
			row, monto, divisa,
			cfg1, cfg2, tipoLetra, signo1, signo2, socio2 != nil,
		)
		if !liquidado || !lado1.ok {
			tasa1, monto1, me1 = liveTasa1, liveMonto1, liveME1
		}
		if !liquidado || (socio2 != nil && !lado2.ok) {
			tasa2, monto2, me2 = liveTasa2, liveMonto2, liveME2
		}
		if tipo1 == "" {
			tipo1 = tipoTag
		}
		if tipo2 == "" {
			tipo2 = tipoTag
		}
	}
	return domain.Comprobante{
		HashLargo: row.HashLargo, HashCorto: row.HashCorto,
		FechaHoraComprobante: row.FechaHora, TimestampMsg: row.TimestampMsg,
		Monto: monto, Moneda: row.Moneda, Banco: row.Banco, Titular: row.Titular,
		Referencia: row.Referencia, ProcesadoIA: row.ProcesadoIA,
		URLImagen: row.URLImagen, Caption: row.Caption,
		NombreSocio1: socio1, MonedaSocio1: monedaSocio1,
		TipoOp1: tipo1, Monto1: monto1, Tasa1: tasa1, ME1: me1,
		NombreSocio2: socio2, MonedaSocio2: monedaSocio2,
		TipoOp2: tipo2, Monto2: monto2, Tasa2: tasa2, ME2: me2,
		M1Socio: monto1, M2Socio: monto2, LoteTasaAsignado: assignedLot(row),
	}
}

// frozenLado conserva los tres valores contables de un lado cuando el snapshot
// los trae completos. Un puntero nil significa que la fila histórica no tiene
// ese dato y que la lectura debe resolverlo en vivo.
type frozenLado struct {
	tasa  float64
	monto float64
	me    float64
	ok    bool
}

func frozenSide(tasa, monto, me *float64) frozenLado {
	if tasa == nil || monto == nil || me == nil {
		return frozenLado{}
	}
	return frozenLado{tasa: *tasa, monto: *monto, me: *me, ok: true}
}

// frozenType devuelve la etiqueta almacenada y usa la de respaldo cuando la
// principal está vacía. Las etiquetas del snapshot ya incluyen la divisa
// (por ejemplo, P-COP).
func frozenType(primary, fallback *string) string {
	for _, candidate := range []*string{primary, fallback} {
		if candidate == nil {
			continue
		}
		if value := strings.TrimSpace(*candidate); value != "" {
			return value
		}
	}
	return ""
}

func resolveSocio1(row SourceRow, liquidado bool) string {
	if liquidado && row.Socio1 != nil && strings.TrimSpace(*row.Socio1) != "" {
		return strings.TrimSpace(*row.Socio1)
	}
	if !liquidado && strings.TrimSpace(row.FBSocio1) != "" {
		return strings.TrimSpace(row.FBSocio1)
	}
	return "GENERAL"
}

func resolveSocio2(row SourceRow, liquidado bool) *string {
	value := row.FBSocio2
	if liquidado {
		value = row.Socio2
	}
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func operationLetter(row SourceRow, conf CurrencyConfig, socio1, socio2, divisa string) string {
	// La letra de una operación que no es abono no se toma de comprobantes_liq:
	// el contrato vivo del socio 1 para la divisa es la única fuente de verdad.
	// Así una fila histórica con P-PEN no puede seguir pintando Pago cuando el
	// perfil ya configura D. Solo un abono almacenado conserva la A; los
	// proveedores nunca la reciben porque su contrapartida la fija el contrato.
	candidata := tipoContractual(conf)
	if !esRolProveedor(row.RolSocio1) && savedLetter(row.TipoOp1) == "A" {
		candidata = "A"
	}
	// La regla de IRIS en COP se aplica después: un A sin caption explícito
	// vuelve al tipo contractual y a sus polar_D / polar_P.
	return NatureForAbono(candidata, socio1, socio2, divisa, captionOf(row), conf)
}

func esRolProveedor(rol string) bool {
	return domain.NormalizeCurrency(rol, "SOCIO") == "PROVEEDOR"
}

func captionOf(row SourceRow) string {
	if row.Caption == nil {
		return ""
	}
	return *row.Caption
}

func savedLetter(value *string) string {
	if value == nil {
		return ""
	}
	return domain.NormalizeCurrency(strings.Split(*value, "-")[0], "")
}

func assignedLot(row SourceRow) string {
	for _, candidate := range []*string{row.LoteTasa, row.LoteTasaHistorico} {
		if candidate != nil && strings.TrimSpace(*candidate) != "" {
			return strings.TrimSpace(*candidate)
		}
	}
	return "T001"
}
