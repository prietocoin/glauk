package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"glauk/internal/comprobantes/domain"
)

// FiltrosReporte alimenta roles, entidades y hashes del modal de Reportes.
func (r *Repository) FiltrosReporte(ctx context.Context, rol string) (domain.ReportFilters, error) {
	normalized := normalizeRole(rol)
	roles, err := r.strings(ctx, reportRolesSQL)
	if err != nil {
		return domain.ReportFilters{}, fmt.Errorf("consultar roles: %w", err)
	}
	entities, err := r.strings(ctx, reportEntitiesSQL, normalized)
	if err != nil {
		return domain.ReportFilters{}, fmt.Errorf("consultar entidades: %w", err)
	}
	hashes, err := r.strings(ctx, reportHashesSQL)
	if err != nil {
		return domain.ReportFilters{}, fmt.Errorf("consultar hashes: %w", err)
	}
	return domain.ReportFilters{Roles: roles, Entidades: entities, Hashes: hashes}, nil
}

// MovimientosReporte devuelve filas ya proyectadas, sin SQL fuera del dominio.
func (r *Repository) MovimientosReporte(ctx context.Context, q domain.ReportQuery) ([]domain.ReportRow, error) {
	q.Normalize()
	query := reportMovementsSQL + orderDirection(q.Orden) + " LIMIT $6"
	rows, err := r.db.Query(ctx, query, append(reportArgs(q), q.Limite)...)
	if err != nil {
		return nil, fmt.Errorf("consultar movimientos: %w", err)
	}
	defer rows.Close()
	items := make([]domain.ReportRow, 0)
	for rows.Next() {
		var row domain.ReportRow
		if err := rows.Scan(&row.HashLargo, &row.HashCorto, &row.Fecha, &row.Socio,
			&row.Rol, &row.Moneda, &row.Monto, &row.TipoOp, &row.Banco,
			&row.Referencia, &row.Estado); err != nil {
			return nil, fmt.Errorf("leer movimientos: %w", err)
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("leer movimientos: %w", err)
	}
	return items, nil
}

// ResumenReporte agrupa por moneda y conserva las fechas efectivas del filtro.
func (r *Repository) ResumenReporte(ctx context.Context, q domain.ReportQuery) (domain.ReportSummary, error) {
	q.Normalize()
	rows, err := r.db.Query(ctx, reportSummarySQL, reportArgs(q)...)
	if err != nil {
		return domain.ReportSummary{}, fmt.Errorf("consultar resumen: %w", err)
	}
	defer rows.Close()
	summary := domain.ReportSummary{PorMoneda: map[string]float64{}}
	for rows.Next() {
		var currency string
		var total int
		var amount float64
		var desde, hasta time.Time
		if err := rows.Scan(&currency, &total, &amount, &desde, &hasta); err != nil {
			return domain.ReportSummary{}, fmt.Errorf("leer resumen: %w", err)
		}
		summary.Total += total
		summary.MontoTotal += amount
		summary.PorMoneda[currency] = amount
		if summary.Desde.IsZero() || desde.Before(summary.Desde) {
			summary.Desde = desde
		}
		if hasta.After(summary.Hasta) {
			summary.Hasta = hasta
		}
	}
	if err := rows.Err(); err != nil {
		return domain.ReportSummary{}, fmt.Errorf("leer resumen: %w", err)
	}
	return summary, nil
}

func (r *Repository) strings(ctx context.Context, query string, args ...any) ([]string, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]string, 0)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func reportArgs(q domain.ReportQuery) []any {
	var desde, hasta any
	if !q.Desde.IsZero() {
		desde = q.Desde
	}
	if !q.Hasta.IsZero() {
		hasta = q.Hasta
	}
	return []any{q.Rol, q.Entidad, q.Hash, desde, hasta}
}

func orderDirection(orden string) string {
	if orden == "fecha_asc" {
		return "ASC"
	}
	return "DESC"
}

func normalizeRole(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.EqualFold(trimmed, "TODOS") {
		return ""
	}
	return trimmed
}
