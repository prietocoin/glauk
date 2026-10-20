package service

import (
	"context"

	"glauk/internal/tasas/store"
)

// Notificacion identifica el lote que se acaba de notificar al flujo de remesas.
type Notificacion struct {
	IDTasa string `json:"id_tasa"`
}

// NotificarRemesas registra el lote vigente en notificaciones_tasas. El disparador
// de remesas cuelga de la tabla; este método solo deja la fila y confirma el ID.
func (s *Service) NotificarRemesas(ctx context.Context) (Notificacion, error) {
	lote, err := s.LoteVigente(ctx)
	if err != nil {
		return Notificacion{}, err
	}
	id := lote.IDTasa
	if id == "" {
		id = "T000"
	}
	notificado, err := store.RecordNotification(ctx, s.pool, id)
	if err != nil {
		return Notificacion{}, err
	}
	return Notificacion{IDTasa: notificado}, nil
}
