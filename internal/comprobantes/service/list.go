package service

import (
	"context"

	"glauk/internal/comprobantes/calc"
	"glauk/internal/comprobantes/domain"
)

// Service es la fachada de casos de uso del dominio Comprobantes.
//
// No escribe SQL ni conoce Fiber: coordina el repositorio, normaliza filtros y
// aplica las reglas de validación de los DTO.
type Service struct {
	repo Repository
}

// New construye el servicio con el puerto de persistencia ya cableado.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// List devuelve la auditoría proyectada y nunca un slice nulo.
func (s *Service) List(ctx context.Context, filters domain.ListFilters) ([]domain.Comprobante, error) {
	filters.Normalize()
	if err := filters.Validate(); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListRows(ctx, filters)
	if err != nil {
		return nil, err
	}
	fundda, err := s.repo.Fundda(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]domain.Comprobante, 0, len(rows))
	for _, row := range rows {
		items = append(items, calc.ProjectRow(row, fundda))
	}
	return items, nil
}
