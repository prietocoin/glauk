package service

import (
	"context"

	"glauk/internal/comprobantes/domain"
)

// ReportFilters, ReportQuery, ReportRow y ReportSummary son los tipos que
// consume el adaptador de Reportes sin conocer SQL ni tablas.
type (
	ReportFilters = domain.ReportFilters
	ReportQuery   = domain.ReportQuery
	ReportRow     = domain.ReportRow
	ReportSummary = domain.ReportSummary
)

// ReportReader es el puerto propio de Comprobantes que consume el adaptador
// MovimientosComprobantes descrito en §3.7.2.
type ReportReader interface {
	FiltrosReporte(ctx context.Context, rol string) (ReportFilters, error)
	MovimientosReporte(ctx context.Context, q ReportQuery) ([]ReportRow, error)
	ResumenReporte(ctx context.Context, q ReportQuery) (ReportSummary, error)
}

var _ ReportReader = (*Service)(nil)

// FiltrosReporte expone roles, entidades y hashes del historial.
func (s *Service) FiltrosReporte(ctx context.Context, rol string) (ReportFilters, error) {
	return s.repo.FiltrosReporte(ctx, rol)
}

// MovimientosReporte proyecta las filas del listado para Reportes.
func (s *Service) MovimientosReporte(ctx context.Context, q ReportQuery) ([]ReportRow, error) {
	return s.repo.MovimientosReporte(ctx, q)
}

// ResumenReporte agrega los movimientos filtrados por moneda.
func (s *Service) ResumenReporte(ctx context.Context, q ReportQuery) (ReportSummary, error) {
	return s.repo.ResumenReporte(ctx, q)
}
