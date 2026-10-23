package calc

import (
	"math"
	"strings"

	"glauk/internal/comprobantes/domain"
)

// Raw es la entrada mínima del snapshot financiero. Caption conserva el texto
// original del comprobante: la regla de abono explícito de IRIS en COP depende
// de él y no puede reconstruirse desde la extracción de la imagen.
type Raw struct {
	HashLargo  string
	Monto      float64
	Moneda     string
	TipoManual string
	Caption    string
	IDTasa     string
}

// RateLot es el lote de tasas congelado o vigente.
type RateLot struct {
	IDTasa string
	Tasas  map[string]float64
}

// CalcularSnapshotFinanciero replica exactamente el congelado legacy.
//
// Aplica abono imperativo, herencia de FUNDDA, ancla de polaridad, spread,
// defaults de tasas y truncado adaptativo. Cualquier cambio aquí exige un
// fixture dorado nuevo en snapshot_test.go.
func CalcularSnapshotFinanciero(raw Raw, socio1, socio2 Profile, lote RateLot,
	fundda Profile) domain.Liquidacion {
	montoRaw := math.Abs(raw.Monto)
	divisa := domain.NormalizeCurrency(raw.Moneda, "USDT")
	tipoOp := OperationNature(raw.TipoManual)
	monedaSocio1 := domain.NormalizeCurrency(socio1.MonedaBase, "USDT")
	if monedaSocio1 == divisa && !IsGeneral(socio1.Nombre) &&
		strings.TrimSpace(socio1.Nombre) != "" {
		tipoOp = "A"
	}
	loteCodigo := domain.FirstNonEmpty(raw.IDTasa, lote.IDTasa, "T052")
	tasas := MergeRates(lote.Tasas)
	baseDivisa := SnapshotRate(tasas, divisa)
	cfg1 := ResolveSocioConfig(socio1, divisa, fundda)
	cfg2 := ResolveSocioConfig(socio2, divisa, fundda)
	// El abono automático por coincidencia de moneda base se revisa antes de
	// calcular signos: IRIS en COP solo conserva la A si el caption la declara.
	tipoOp = NatureForAbono(tipoOp, socio1.Nombre, socio2.Nombre, divisa, raw.Caption, cfg1.Conf)
	signo1, signo2 := SideSigns(cfg1, cfg2, tipoOp, socio1.Nombre)
	lado1 := CalculateSide(cfg1, signo1, tipoOp, montoRaw, baseDivisa, tasas)
	lado2 := CalculateSide(cfg2, signo2, tipoOp, montoRaw, baseDivisa, tasas)
	etiqueta := tipoOp + "-" + divisa
	socio1Nombre := strings.TrimSpace(socio1.Nombre)
	if socio1Nombre == "" {
		socio1Nombre = "GENERAL"
	}
	return domain.Liquidacion{
		HashLargo: strings.TrimSpace(raw.HashLargo), Socio1: socio1Nombre,
		TipoOp1: etiqueta, Monto1: lado1.monto, Tasa1: lado1.tasa, ME1: lado1.me,
		Socio2: OptionalSocio(socio2.Nombre), TipoOp2: etiqueta,
		Monto2: lado2.monto, Tasa2: lado2.tasa, ME2: lado2.me, LoteTasa: loteCodigo,
	}
}
