package service

import (
	"context"

	"glauk/internal/comprobantes/domain"
)

// Latest devuelve el lote vigente o el fallback T001 del legado.
//
// La ruta pública nunca responde 404 por falta de tasas: el panel necesita
// siempre un conjunto de divisas para pintar el formulario de liquidación.
func (s *Service) Latest(ctx context.Context) (domain.LoteTasa, error) {
	return s.repo.LatestLot(ctx)
}

// History devuelve los últimos veinte lotes, posiblemente vacío.
//
// El contrato HTTP exige un array raíz; por eso un nil del repositorio se
// convierte aquí en un slice vacío antes de cruzar la frontera de transporte.
func (s *Service) History(ctx context.Context) ([]domain.LoteTasa, error) {
	lots, err := s.repo.History(ctx)
	if err != nil {
		return nil, err
	}
	if lots == nil {
		return []domain.LoteTasa{}, nil
	}
	return lots, nil
}
