package calc

import "glauk/internal/comprobantes/domain"

type snapshotSide struct {
	monto float64
	me    float64
	tasa  float64
}

// CalculateSide aplica spread, tasa efectiva y truncado de un lado.
func CalculateSide(cfg SocioConfig, signo int, tipoOp string, monto, baseDivisa float64,
	tasas map[string]float64) snapshotSide {
	if !cfg.Active {
		return snapshotSide{tasa: 1}
	}
	factor := SpreadFactor(cfg.Conf, tipoOp)
	baseSocio := SnapshotRate(tasas, cfg.Moneda)
	divisor := domain.PositiveOrOne(baseSocio)
	tasa := TruncateSnapshotRate((baseDivisa / divisor) * factor)
	divisorTasa := domain.PositiveOrOne(tasa)
	montoNominal := TruncateAmount(float64(signo) * (monto / divisorTasa))
	me := TruncateAmount(montoNominal / divisor)
	return snapshotSide{monto: montoNominal, me: me, tasa: tasa}
}

// SpreadFactor aplica el porcentaje de depósito o de pago según la operación.
func SpreadFactor(conf CurrencyConfig, tipoLetra string) float64 {
	deposito := AbsPercent(conf.Porcentaje.Deposito)
	pago := AbsPercent(conf.Porcentaje.Pago)
	if tipoLetra == "D" || tipoLetra == "A" {
		return 1 + deposito/100
	}
	return 1 - pago/100
}

// SignedDivision divide sin perder el signo y evita tasas no positivas.
func SignedDivision(signo int, monto, tasa float64) float64 {
	if tasa > 0 {
		return float64(signo) * monto / tasa
	}
	return float64(signo) * monto
}

// CalculateAmounts proyecta una fila live o devuelve el snapshot congelado.
func CalculateAmounts(row SourceRow, liquidado bool, monto float64,
	divisa, monedaSocio1, monedaSocio2 string,
	conf1, conf2 CurrencyConfig, tipoLetra string,
	signo1, signo2 int, hasSocio2 bool) (float64, float64, float64, float64, float64, float64) {
	if liquidado {
		return FrozenAmounts(row, signo1, signo2)
	}
	tasas := DecodeRates(row.TasasLote)
	tasaBaseDivisa := RateOr(tasas, divisa)
	tasaBase1 := RateOr(tasas, monedaSocio1)
	tasaBase2 := RateOr(tasas, monedaSocio2)
	tasa1 := TruncateCommercialRate((tasaBaseDivisa / domain.PositiveOrOne(tasaBase1)) *
		SpreadFactor(conf1, tipoLetra))
	tasa2 := TruncateCommercialRate((tasaBaseDivisa / domain.PositiveOrOne(tasaBase2)) *
		SpreadFactor(conf2, tipoLetra))
	monto1 := SignedDivision(signo1, monto, tasa1)
	me1 := monto1 / domain.PositiveOrOne(tasaBase1)
	if !hasSocio2 {
		return tasa1, monto1, me1, tasa2, 0, 0
	}
	monto2 := SignedDivision(signo2, monto, tasa2)
	me2 := monto2 / domain.PositiveOrOne(tasaBase2)
	return tasa1, monto1, me1, tasa2, monto2, me2
}

// FrozenAmounts reaplica polaridad a las magnitudes ya congeladas.
func FrozenAmounts(row SourceRow, signo1, signo2 int) (float64, float64, float64, float64, float64, float64) {
	tasa1 := TasaOrOne(row.Tasa1)
	tasa2 := TasaOrOne(row.Tasa2)
	monto1 := float64(signo1) * absValue(ValueOr(row.Monto1, 0))
	me1 := float64(signo1) * absValue(ValueOr(row.ME1, 0))
	monto2 := float64(signo2) * absValue(ValueOr(row.Monto2, 0))
	me2 := float64(signo2) * absValue(ValueOr(row.ME2, 0))
	return tasa1, monto1, me1, tasa2, monto2, me2
}

func absValue(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
