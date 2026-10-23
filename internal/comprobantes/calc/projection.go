package calc

import (
	"math"
	"strings"

	"glauk/internal/comprobantes/domain"
)

// ProjectRow transforma una fila cruda en el contrato de auditoría.
//
// Si la fila está liquidada, los importes congelados mandan y no se vuelven a
// firmar. Solo las filas live aplican la polaridad del contrato y el ancla de
// herencia, con la misma función que usa el snapshot.
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
	profile1 := DecodeProfile(row.Socio1Row)
	profile2 := DecodeProfile(row.Socio2Row)
	cfg1 := ResolveLiveSocioConfig(socio1, profile1, divisa, fundda, monedaSocio1)
	cfg2 := ResolveLiveSocioConfig(socio2Nombre, profile2, divisa, fundda, monedaSocio2)
	tipoLetra := operationLetter(row, cfg1.Conf, socio1, monedaSocio1, divisa)
	tipoTag := tipoLetra + "-" + divisa
	signo1, signo2 := 0, 0
	if !liquidado {
		signo1, signo2 = SideSigns(cfg1, cfg2, tipoLetra)
	}
	tasa1, monto1, me1, tasa2, monto2, me2 := CalculateAmounts(
		row, liquidado, monto, divisa,
		cfg1, cfg2, tipoLetra, signo1, signo2, socio2 != nil,
	)
	return domain.Comprobante{
		HashLargo: row.HashLargo, HashCorto: row.HashCorto,
		FechaHoraComprobante: row.FechaHora, TimestampMsg: row.TimestampMsg,
		Monto: monto, Moneda: row.Moneda, Banco: row.Banco, Titular: row.Titular,
		Referencia: row.Referencia, ProcesadoIA: row.ProcesadoIA,
		URLImagen: row.URLImagen, Caption: row.Caption,
		NombreSocio1: socio1, MonedaSocio1: monedaSocio1,
		TipoOp1: tipoTag, Monto1: monto1, Tasa1: tasa1, ME1: me1,
		NombreSocio2: socio2, MonedaSocio2: monedaSocio2,
		TipoOp2: tipoTag, Monto2: monto2, Tasa2: tasa2, ME2: me2,
		M1Socio: monto1, M2Socio: monto2, LoteTasaAsignado: assignedLot(row),
	}
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

func operationLetter(row SourceRow, conf CurrencyConfig, socio1, monedaSocio1, divisa string) string {
	rol := domain.NormalizeCurrency(row.RolSocio1, "SOCIO")
	guardada := savedLetter(row.TipoOp1)
	if rol == "PROVEEDOR" {
		if guardada != "" && guardada != "A" {
			return guardada
		}
		return domain.NormalizeCurrency(conf.Tipo, "D")
	}
	if guardada != "" {
		return guardada
	}
	if monedaSocio1 == divisa && socio1 != "GENERAL" {
		return "A"
	}
	return domain.NormalizeCurrency(conf.Tipo, "D")
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
