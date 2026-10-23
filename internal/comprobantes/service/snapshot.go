package service

import (
	"context"

	"glauk/internal/comprobantes/calc"
	"glauk/internal/comprobantes/domain"
	"glauk/internal/comprobantes/dto"
	"glauk/internal/comprobantes/store"
)

// buildSnapshot recalcula el congelado con las lecturas de la misma transacción.
func (s *Service) buildSnapshot(ctx context.Context, tx store.Tx, hash string,
	update store.RawUpdate, input dto.UpdateInput) (domain.Liquidacion, error) {
	loteID := input.Lote()
	lote, err := tx.RateLotByID(ctx, loteID)
	if err != nil {
		return domain.Liquidacion{}, err
	}
	if lote.IDTasa == "" {
		lote, err = tx.LatestRateLot(ctx)
		if err != nil {
			return domain.Liquidacion{}, err
		}
	}
	socio1, err := tx.ProfileByName(ctx, input.Socio1Nombre())
	if err != nil {
		return domain.Liquidacion{}, err
	}
	socio2, err := tx.ProfileByName(ctx, input.Socio2Nombre())
	if err != nil {
		return domain.Liquidacion{}, err
	}
	fundda, err := tx.Fundda(ctx)
	if err != nil {
		return domain.Liquidacion{}, err
	}

	moneda := domain.NormalizeCurrency(update.Moneda, "COP")
	tipo := input.TipoValor()
	liq := calc.CalcularSnapshotFinanciero(calc.Raw{
		HashLargo: hash, Monto: update.Monto, Moneda: moneda,
		TipoManual: tipo, IDTasa: loteID,
	}, socio1, socio2, lote, fundda)

	etiqueta := calc.OperationNature(tipo) + "-" + moneda
	liq.TipoOp1, liq.TipoOp2 = etiqueta, etiqueta
	tasas := calc.MergeRates(lote.Tasas)
	base1 := calc.SnapshotRate(tasas, calc.CurrencyOf(socio1.MonedaBase))
	base2 := calc.SnapshotRate(tasas, calc.CurrencyOf(socio2.MonedaBase))
	if value, ok := input.Monto1Valor(); ok {
		// El override manual cambia el nominal; el ME debe seguirlo con la
		// misma tasa base que usó el snapshot para no quedar incoherente.
		liq.Monto1 = value
		liq.ME1 = calc.TruncateAmount(value / base1)
	}
	if value, ok := input.Monto2Valor(); ok {
		liq.Monto2 = value
		liq.ME2 = calc.TruncateAmount(value / base2)
	}
	return liq, nil
}
