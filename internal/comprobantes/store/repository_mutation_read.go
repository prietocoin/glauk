package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"glauk/internal/comprobantes/calc"
	"glauk/internal/comprobantes/domain"
)

// RawComprobante es la fila bloqueada antes de editar.
type RawComprobante struct {
	Monto      float64
	Moneda     *string
	Banco      *string
	Referencia *string
	Titular    *string
	// Caption alimenta la regla de abono explícito de IRIS en COP: la
	// recalculación de una edición debe ver el mismo texto que vio la ingesta.
	Caption string
}

// RawUpdate contiene los valores que se escribirán en comprobantes_raw.
type RawUpdate struct {
	Monto      float64
	Moneda     string
	Banco      string
	Referencia string
	Titular    string
}

const lockComprobanteSQL = `
SELECT COALESCE(cr.monto, 0), cr.moneda, cr.banco, cr.referencia, cr.titular,
       COALESCE((
           SELECT i.caption
           FROM impactos_raw i
           WHERE LOWER(TRIM(i.hash_largo)) = LOWER(TRIM(cr.hash_largo))
             AND i.eliminado_en IS NULL
           ORDER BY i.timestamp_msg DESC NULLS LAST, i.id DESC
           LIMIT 1
       ), '')
FROM comprobantes_raw cr
WHERE LOWER(TRIM(cr.hash_largo)) = LOWER(TRIM($1))
FOR UPDATE`

const profileByNameSQL = `
SELECT to_jsonb(p) FROM perfiles_glaukov p
WHERE UPPER(TRIM(p.nombre)) = UPPER(TRIM($1)) LIMIT 1`

const profileExistsSQL = `
SELECT EXISTS (
    SELECT 1 FROM perfiles_glaukov p
    WHERE UPPER(TRIM(p.nombre)) = UPPER(TRIM($1))
)`

const funddaTxSQL = `
SELECT to_jsonb(p) FROM perfiles_glaukov p
WHERE UPPER(TRIM(p.nombre)) = 'FUNDDA' LIMIT 1`

const rateLotByIDSQL = `
SELECT id_tasa, tasas FROM tasas_glaukov
WHERE UPPER(TRIM(id_tasa)) = UPPER(TRIM($1))
ORDER BY created_at DESC, id DESC LIMIT 1`

const latestRateLotSQL = `
SELECT id_tasa, tasas FROM tasas_glaukov
ORDER BY created_at DESC, id DESC LIMIT 1`

const currentLiquidacionSQL = `
SELECT socio_1, tipo_op1, monto_1, tasa_1, me1,
       socio_2, tipo_op2, monto_2, tasa_2, me2, lote_tasa
FROM comprobantes_liq
WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($1))`

type calcProfile = calc.Profile
type calcRateLot = calc.RateLot

// LiquidacionActual devuelve el snapshot vigente para que una edición pueda
// conservar socio, tipo, lote e importes cuando el payload solo cambia otros
// campos.
func (t *txStore) LiquidacionActual(ctx context.Context, hash string) (domain.Liquidacion, bool, error) {
	var liq domain.Liquidacion
	var socio1, tipo1, socio2, tipo2, lote *string
	var monto1, tasa1, me1, monto2, tasa2, me2 *float64
	err := t.tx.QueryRow(ctx, currentLiquidacionSQL, hash).Scan(
		&socio1, &tipo1, &monto1, &tasa1, &me1,
		&socio2, &tipo2, &monto2, &tasa2, &me2, &lote,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return liq, false, nil
	}
	if err != nil {
		return liq, false, fmt.Errorf("consultar liquidación actual: %w", err)
	}
	liq.HashLargo = strings.TrimSpace(hash)
	liq.Socio1 = trimmedOrEmpty(socio1)
	liq.TipoOp1 = trimmedOrEmpty(tipo1)
	liq.Monto1 = floatOrZero(monto1)
	liq.Tasa1 = floatOrZero(tasa1)
	liq.ME1 = floatOrZero(me1)
	liq.TipoOp2 = trimmedOrEmpty(tipo2)
	liq.Monto2 = floatOrZero(monto2)
	liq.Tasa2 = floatOrZero(tasa2)
	liq.ME2 = floatOrZero(me2)
	liq.LoteTasa = trimmedOrEmpty(lote)
	if socio2 != nil {
		value := strings.TrimSpace(*socio2)
		liq.Socio2 = &value
	}
	return liq, true, nil
}

func trimmedOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func floatOrZero(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

// LockComprobante bloquea la fila base y devuelve sus valores actuales.
func (t *txStore) LockComprobante(ctx context.Context, hash string) (RawComprobante, error) {
	var row RawComprobante
	err := t.tx.QueryRow(ctx, lockComprobanteSQL, hash).Scan(
		&row.Monto, &row.Moneda, &row.Banco, &row.Referencia, &row.Titular, &row.Caption,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return row, domain.ErrComprobanteNotFound
	}
	if err != nil {
		return row, fmt.Errorf("bloquear comprobante: %w", err)
	}
	return row, nil
}

// UpdateComprobante escribe los campos base ya resueltos por el servicio.
func (t *txStore) UpdateComprobante(ctx context.Context, hash string, data RawUpdate) error {
	_, err := t.tx.Exec(ctx, updateRawSQL,
		data.Monto, data.Moneda, data.Banco, data.Referencia, data.Titular, hash,
	)
	if err != nil {
		return fmt.Errorf("actualizar comprobante: %w", err)
	}
	return nil
}

// ProfileByName devuelve el perfil real. GENERAL, los placeholders y los
// nombres no registrados se rechazan: no son una contraparte contable.
func (t *txStore) ProfileByName(ctx context.Context, name string) (calc.Profile, error) {
	trimmed := strings.TrimSpace(name)
	if domain.IsPlaceholderSocio(trimmed) {
		return calc.Profile{}, domain.Validationf("el socio %q no es un socio real", name)
	}
	var raw []byte
	err := t.tx.QueryRow(ctx, profileByNameSQL, trimmed).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return calc.Profile{}, domain.Validationf("el socio %q no está registrado en perfiles_glaukov", name)
	}
	if err != nil {
		return calc.Profile{}, fmt.Errorf("consultar perfil %s: %w", trimmed, err)
	}
	return calc.DecodeProfile(raw), nil
}

// ProfilesExist valida que ambos nombres correspondan a perfiles reales antes
// de aceptar una liquidación manual.
func (r *Repository) ProfilesExist(ctx context.Context, names ...string) (bool, error) {
	for _, name := range names {
		if domain.IsPlaceholderSocio(name) {
			return false, nil
		}
		var exists bool
		if err := r.db.QueryRow(ctx, profileExistsSQL, name).Scan(&exists); err != nil {
			return false, fmt.Errorf("consultar perfil %s: %w", name, err)
		}
		if !exists {
			return false, nil
		}
	}
	return true, nil
}

// Fundda lee la plantilla de herencia dentro de la transacción.
func (t *txStore) Fundda(ctx context.Context) (calc.Profile, error) {
	return scanProfile(t.tx.QueryRow(ctx, funddaTxSQL), "FUNDDA")
}

// RateLotByID devuelve el lote pedido o un valor vacío si no existe.
func (t *txStore) RateLotByID(ctx context.Context, id string) (calc.RateLot, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return calc.RateLot{}, nil
	}
	var lot calc.RateLot
	var raw []byte
	err := t.tx.QueryRow(ctx, rateLotByIDSQL, trimmed).Scan(&lot.IDTasa, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return calc.RateLot{}, nil
	}
	if err != nil {
		return calc.RateLot{}, fmt.Errorf("consultar lote de tasas %s: %w", trimmed, err)
	}
	lot.Tasas = calc.DecodeRates(raw)
	return lot, nil
}

// LatestRateLot devuelve el lote más reciente dentro de la transacción.
func (t *txStore) LatestRateLot(ctx context.Context) (calc.RateLot, error) {
	var lot calc.RateLot
	var raw []byte
	err := t.tx.QueryRow(ctx, latestRateLotSQL).Scan(&lot.IDTasa, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return calc.RateLot{IDTasa: "T001", Tasas: calc.DefaultRates()}, nil
	}
	if err != nil {
		return calc.RateLot{}, fmt.Errorf("consultar últimas tasas: %w", err)
	}
	lot.Tasas = calc.DecodeRates(raw)
	return lot, nil
}

func scanProfile(row pgx.Row, label string) (calc.Profile, error) {
	var raw []byte
	err := row.Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return calc.Profile{}, nil
	}
	if err != nil {
		return calc.Profile{}, fmt.Errorf("consultar perfil %s: %w", label, err)
	}
	return calc.DecodeProfile(raw), nil
}
