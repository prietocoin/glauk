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
SELECT COALESCE(monto, 0), moneda, banco, referencia, titular
FROM comprobantes_raw
WHERE LOWER(TRIM(hash_largo)) = LOWER(TRIM($1))
FOR UPDATE`

const profileByNameSQL = `
SELECT to_jsonb(p) FROM perfiles_glaukov p
WHERE UPPER(TRIM(p.nombre)) = UPPER(TRIM($1)) LIMIT 1`

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

type calcProfile = calc.Profile
type calcRateLot = calc.RateLot

// LockComprobante bloquea la fila base y devuelve sus valores actuales.
func (t *txStore) LockComprobante(ctx context.Context, hash string) (RawComprobante, error) {
	var row RawComprobante
	err := t.tx.QueryRow(ctx, lockComprobanteSQL, hash).Scan(
		&row.Monto, &row.Moneda, &row.Banco, &row.Referencia, &row.Titular,
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

// ProfileByName devuelve GENERAL sin consultar y tolera socios inexistentes.
func (t *txStore) ProfileByName(ctx context.Context, name string) (calc.Profile, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || strings.EqualFold(trimmed, "GENERAL") {
		return calc.Profile{Nombre: "GENERAL", MonedaBase: "USDT"}, nil
	}
	var raw []byte
	err := t.tx.QueryRow(ctx, profileByNameSQL, trimmed).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return calc.Profile{Nombre: trimmed, MonedaBase: "USDT"}, nil
	}
	if err != nil {
		return calc.Profile{}, fmt.Errorf("consultar perfil %s: %w", trimmed, err)
	}
	return calc.DecodeProfile(raw), nil
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
