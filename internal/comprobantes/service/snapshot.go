package service

import (
	"context"
	"strings"

	"glauk/internal/comprobantes/calc"
	"glauk/internal/comprobantes/domain"
	"glauk/internal/comprobantes/dto"
	"glauk/internal/comprobantes/store"
)

// buildSnapshot recalcula el snapshot financiero completo con los perfiles
// vigentes dentro de la misma transacción.
//
// Un payload parcial conserva el socio, tipo y lote ya persistidos: editar el
// monto o el banco no debe degradar la liquidación a GENERAL ni cambiar el lote.
func (s *Service) buildSnapshot(ctx context.Context, tx store.Tx, hash string,
	update store.RawUpdate, input dto.UpdateInput,
	actual domain.Liquidacion, caption string) (domain.Liquidacion, error) {
	loteID := actual.LoteTasa
	if value, ok := input.LoteOpcional(); ok {
		loteID = value
	}
	if strings.TrimSpace(loteID) == "" {
		loteID = dto.DefaultLoteTasa
	}
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
	socio1Nombre := actual.Socio1
	if value, ok := input.Socio1Opcional(); ok {
		socio1Nombre = value
	}
	if strings.TrimSpace(socio1Nombre) == "" {
		socio1Nombre = dto.DefaultSocio
	}
	socio2Nombre := ""
	if actual.Socio2 != nil {
		socio2Nombre = strings.TrimSpace(*actual.Socio2)
	}
	if value, ok := input.Socio2Opcional(); ok {
		socio2Nombre = value
	}
	if strings.TrimSpace(socio2Nombre) == "" {
		socio2Nombre = dto.DefaultSocio
	}
	tipo := actual.TipoOp1
	if value, ok := input.TipoOpcional(); ok {
		tipo = value
	}
	if strings.TrimSpace(tipo) == "" {
		tipo = dto.DefaultTipo
	}

	socio1, err := tx.ProfileByName(ctx, socio1Nombre)
	if err != nil {
		return domain.Liquidacion{}, err
	}
	socio2, err := tx.ProfileByName(ctx, socio2Nombre)
	if err != nil {
		return domain.Liquidacion{}, err
	}
	fundda, err := tx.Fundda(ctx)
	if err != nil {
		return domain.Liquidacion{}, err
	}

	moneda := domain.NormalizeCurrency(update.Moneda, "COP")
	liq := calc.CalcularSnapshotFinanciero(calc.Raw{
		HashLargo: hash, Monto: update.Monto, Moneda: moneda,
		TipoManual: tipo, Caption: caption, IDTasa: loteID,
	}, socio1, socio2, lote, fundda)

	// El snapshot ya resolvió la naturaleza, incluida la regla de abono
	// explícito de IRIS en COP. Reimponer el tipo de entrada borraría esa
	// decisión y volvería a etiquetar como A un pago sin caption.
	return liq, nil
}
