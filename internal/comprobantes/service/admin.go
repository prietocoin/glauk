package service

import (
	"context"
	"strings"

	"glauk/internal/comprobantes/domain"
	"glauk/internal/comprobantes/dto"
)

// Queue devuelve la consola administrativa sin cachear datos sensibles.
func (s *Service) Queue(ctx context.Context) ([]domain.QueueItem, error) {
	items, err := s.repo.Queue(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []domain.QueueItem{}, nil
	}
	return items, nil
}

// UpdateQueue valida el hash y delega los campos raw ya normalizados.
func (s *Service) UpdateQueue(ctx context.Context, hash string,
	input dto.UpdateInput) (domain.QueueUpdate, error) {
	target := strings.TrimSpace(hash)
	if target == "" {
		return domain.QueueUpdate{}, domain.Validationf("hashLargo es obligatorio")
	}
	update := input.QueueUpdate()
	updated, err := s.repo.UpdateQueue(ctx, target, update)
	if err != nil {
		return domain.QueueUpdate{}, err
	}
	if !updated {
		return domain.QueueUpdate{}, domain.ErrQueueItemNotFound
	}
	return update, nil
}

// DeleteQueue elimina las tablas raw de la consola en una transacción.
func (s *Service) DeleteQueue(ctx context.Context, hash string) error {
	target := strings.TrimSpace(hash)
	if target == "" {
		return domain.Validationf("hashLargo es obligatorio")
	}
	return s.repo.DeleteQueue(ctx, target)
}
