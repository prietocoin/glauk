package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"glauk/internal/comprobantes/calc"
	"glauk/internal/comprobantes/domain"
)

const latestLotSQL = `
SELECT id_tasa, tasas, created_at FROM tasas_glaukov
ORDER BY created_at DESC, id DESC LIMIT 1`

const historyLotsSQL = `
SELECT id_tasa, tasas, created_at FROM tasas_glaukov
ORDER BY created_at DESC, id DESC LIMIT 20`

// LatestLot devuelve el último lote o el fallback T001 sin 404.
func (r *Repository) LatestLot(ctx context.Context) (domain.LoteTasa, error) {
	var lot domain.LoteTasa
	var raw []byte
	var createdAt *time.Time
	err := r.db.QueryRow(ctx, latestLotSQL).Scan(&lot.IDTasa, &raw, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.LoteTasa{IDTasa: "T001", Tasas: calc.DefaultRates()}, nil
	}
	if err != nil {
		return domain.LoteTasa{}, fmt.Errorf("consultar últimas tasas: %w", err)
	}
	lot.Tasas = calc.MergeRates(calc.DecodeRates(raw))
	lot.CreatedAt = createdAt
	return lot, nil
}

// History devuelve los últimos 20 lotes y nunca un slice nulo.
func (r *Repository) History(ctx context.Context) ([]domain.LoteTasa, error) {
	rows, err := r.db.Query(ctx, historyLotsSQL)
	if err != nil {
		return nil, fmt.Errorf("consultar historial de tasas: %w", err)
	}
	defer rows.Close()
	lots := make([]domain.LoteTasa, 0, 20)
	for rows.Next() {
		var lot domain.LoteTasa
		var raw []byte
		var createdAt *time.Time
		if err := rows.Scan(&lot.IDTasa, &raw, &createdAt); err != nil {
			return nil, fmt.Errorf("leer historial de tasas: %w", err)
		}
		lot.Tasas = calc.MergeRates(calc.DecodeRates(raw))
		lot.CreatedAt = createdAt
		lots = append(lots, lot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("leer historial de tasas: %w", err)
	}
	return lots, nil
}
