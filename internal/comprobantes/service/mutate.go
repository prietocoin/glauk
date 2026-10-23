package service

import (
	"context"
	"math"
	"strings"

	"glauk/internal/comprobantes/domain"
	"glauk/internal/comprobantes/dto"
	"glauk/internal/comprobantes/store"
)

// Liquidar registra o actualiza el snapshot congelado de una liquidación.
func (s *Service) Liquidar(ctx context.Context, input dto.LiquidarInput) (domain.Liquidacion, error) {
	liq := input.ToLiquidacion()
	if liq.HashLargo == "" {
		return domain.Liquidacion{}, domain.Validationf("hash_largo es obligatorio")
	}
	if err := validateLiquidacion(liq); err != nil {
		return domain.Liquidacion{}, err
	}
	if err := s.repo.Liquidar(ctx, liq); err != nil {
		return domain.Liquidacion{}, err
	}
	return liq, nil
}

// Update edita comprobantes_raw y sincroniza comprobantes_liq en una transacción.
func (s *Service) Update(ctx context.Context, hash string, input dto.UpdateInput) (domain.Liquidacion, error) {
	target := strings.TrimSpace(hash)
	if target == "" {
		return domain.Liquidacion{}, domain.Validationf("hashLargo es obligatorio")
	}
	var liq domain.Liquidacion
	err := s.repo.InTx(ctx, func(tx store.Tx) error {
		current, err := tx.LockComprobante(ctx, target)
		if err != nil {
			return err
		}
		actual, _, err := tx.LiquidacionActual(ctx, target)
		if err != nil {
			return err
		}
		update := resolveRawUpdate(current, input)
		if err := tx.UpdateComprobante(ctx, target, update); err != nil {
			return err
		}
		liq, err = s.buildSnapshot(ctx, tx, target, update, input, actual, current.Caption)
		if err != nil {
			return err
		}
		return tx.UpsertLiquidacion(ctx, liq)
	})
	return liq, err
}

// Delete borra la fila base y marca impactos_raw dentro de la misma transacción.
// El store aplica el orden canónico raw -> liq -> impactos; no se debe invertir
// desde el servicio para no reintroducir el deadlock de DB-19.
func (s *Service) Delete(ctx context.Context, hash string) error {
	target := strings.TrimSpace(hash)
	if target == "" {
		return domain.Validationf("hashLargo es obligatorio")
	}
	return s.repo.InTx(ctx, func(tx store.Tx) error {
		return tx.DeleteComprobante(ctx, target)
	})
}

// Releer marca el comprobante para que un worker lo procese de nuevo.
func (s *Service) Releer(ctx context.Context, hash string) error {
	target := strings.TrimSpace(hash)
	if target == "" {
		return domain.Validationf("hashLargo es obligatorio")
	}
	updated, err := s.repo.MarkReprocessing(ctx, target)
	if err != nil {
		return err
	}
	if !updated {
		return domain.ErrComprobanteNotFound
	}
	return nil
}

func resolveRawUpdate(current store.RawComprobante, input dto.UpdateInput) store.RawUpdate {
	out := store.RawUpdate{
		Monto: current.Monto, Moneda: derefString(current.Moneda),
		Banco: derefString(current.Banco), Referencia: derefString(current.Referencia),
		Titular: derefString(current.Titular),
	}
	if input.Monto.Set && input.Monto.Value > 0 {
		out.Monto = input.Monto.Value
	}
	if value := strings.TrimSpace(input.MonedaValor()); value != "" {
		out.Moneda = domain.NormalizeCurrency(value, "COP")
	}
	if value := strings.TrimSpace(input.Banco); value != "" {
		out.Banco = strings.ToUpper(value)
	}
	if value := strings.TrimSpace(input.Referencia); value != "" {
		out.Referencia = value
	}
	if value := strings.TrimSpace(input.Titular); value != "" {
		out.Titular = strings.ToUpper(value)
	}
	return out
}

func validateLiquidacion(liq domain.Liquidacion) error {
	values := []float64{liq.Monto1, liq.Tasa1, liq.ME1, liq.Monto2, liq.Tasa2, liq.ME2}
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return domain.Validationf("la liquidación contiene valores no finitos")
		}
	}
	return nil
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
