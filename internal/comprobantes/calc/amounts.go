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
//
// Los signos solo se usan para la rama live. Si la fila ya está liquidada, el
// snapshot persistido es la fuente de verdad y se devuelve sin recalcular.
func CalculateAmounts(row SourceRow, liquidado bool, monto float64, divisa string,
	cfg1, cfg2 SocioConfig, tipoLetra string,
	signo1, signo2 int, hasSocio2 bool) (float64, float64, float64, float64, float64, float64) {
	if liquidado {
		return FrozenAmounts(row)
	}
	tasas := DecodeRates(row.TasasLote)
	tasaBaseDivisa := RateOr(tasas, divisa)
	tasaBase1 := RateOr(tasas, cfg1.Moneda)
	tasaBase2 := RateOr(tasas, cfg2.Moneda)
	tasa1 := TruncateCommercialRate((tasaBaseDivisa / domain.PositiveOrOne(tasaBase1)) *
		SpreadFactor(cfg1.Conf, tipoLetra))
	tasa2 := TruncateCommercialRate((tasaBaseDivisa / domain.PositiveOrOne(tasaBase2)) *
		SpreadFactor(cfg2.Conf, tipoLetra))
	var monto1, me1 float64
	if cfg1.Active {
		monto1 = SignedDivision(signo1, monto, tasa1)
		me1 = monto1 / domain.PositiveOrOne(tasaBase1)
	} else {
		tasa1 = 1
	}
	if !hasSocio2 {
		return tasa1, monto1, me1, 1, 0, 0
	}
	var monto2, me2 float64
	if cfg2.Active {
		monto2 = SignedDivision(signo2, monto, tasa2)
		me2 = monto2 / domain.PositiveOrOne(tasaBase2)
	} else {
		tasa2 = 1
	}
	return tasa1, monto1, me1, tasa2, monto2, me2
}

// FrozenAmounts devuelve los importes congelados tal como los persistió el
// snapshot. El signo almacenado manda: no se aplica valor absoluto ni se
// re-firma con el perfil actual.
func FrozenAmounts(row SourceRow) (float64, float64, float64, float64, float64, float64) {
	return TasaOrOne(row.Tasa1), ValueOr(row.Monto1, 0), ValueOr(row.ME1, 0),
		TasaOrOne(row.Tasa2), ValueOr(row.Monto2, 0), ValueOr(row.ME2, 0)
}
