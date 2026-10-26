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
	if err := validateLiquidacion(liq); err != nil {
		return domain.Liquidacion{}, err
	}
	if ok, err := s.repo.ProfilesExist(ctx, liq.Socio1, *liq.Socio2); err != nil {
		return domain.Liquidacion{}, err
	} else if !ok {
		return domain.Liquidacion{}, domain.Validationf(
			"socio_1 y socio_2 deben existir en perfiles_glaukov")
	}
	if err := s.repo.Liquidar(ctx, liq); err != nil {
		return domain.Liquidacion{}, err
	}
	return liq, nil
}

// Update edita comprobantes_raw y, solo cuando hay un cambio financiero real,
// sincroniza comprobantes_liq en la misma transacción. Una edición de titular,
// banco o referencia no recalcula ni degrada el snapshot.
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
		if !tieneCambiosFinancieros(input, current, actual) {
			liq = actual
			if strings.TrimSpace(liq.HashLargo) == "" {
				liq.HashLargo = target
			}
			return nil
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

// tieneCambiosFinancieros compara el payload con el estado persistido. La UI
// envía el formulario completo, así que la sola presencia de un campo no basta:
// solo se recalcula cuando el valor financiero realmente cambió.
func tieneCambiosFinancieros(input dto.UpdateInput, current store.RawComprobante,
	actual domain.Liquidacion) bool {
	if input.Monto.Set && input.Monto.Value > 0 &&
		!casiIgual(input.Monto.Value, current.Monto) {
		return true
	}
	if value := strings.TrimSpace(input.MonedaValor()); value != "" &&
		!strings.EqualFold(value, derefString(current.Moneda)) {
		return true
	}
	if value, ok := input.Socio1Opcional(); ok && !mismoSocio(value, actual.Socio1) {
		return true
	}
	if value, ok := input.Socio2Opcional(); ok &&
		!mismoSocio(value, derefString(actual.Socio2)) {
		return true
	}
	if value, ok := input.TipoOpcional(); ok &&
		letraTipo(value) != letraTipo(actual.TipoOp1) {
		return true
	}
	if value, ok := input.LoteOpcional(); ok &&
		!strings.EqualFold(value, actual.LoteTasa) {
		return true
	}
	return false
}

func casiIgual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func mismoSocio(a, b string) bool {
	primero := domain.CanonicalSocio(a)
	segundo := domain.CanonicalSocio(b)
	if domain.IsPlaceholderSocio(primero) && domain.IsPlaceholderSocio(segundo) {
		return true
	}
	return primero == segundo
}

func letraTipo(value string) string {
	clean := strings.ToUpper(strings.TrimSpace(value))
	if clean == "" {
		return ""
	}
	return clean[:1]
}

func validateLiquidacion(liq domain.Liquidacion) error {
	return liq.Validate()
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
