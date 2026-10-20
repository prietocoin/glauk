package service

import (
	"context"

	"glauk/internal/tasas/domain"
	"glauk/internal/tasas/store"
)

// Monitor es el lote que muestra el panel. Con la tabla vacía conserva T000 y un
// objeto de tasas vacío: es un estado válido, no un 404.
type Monitor struct {
	IDTasa string         `json:"id_tasa"`
	Tasas  map[string]any `json:"tasas"`
}

// LoteVigente devuelve el último lote guardado o uno vacío cuando todavía no hay
// ninguno. El monitor nunca responde 404: el panel dibuja T000 y espera el primero.
func (s *Service) LoteVigente(ctx context.Context) (domain.Lote, error) {
	lotes, err := store.RecentLotes(ctx, s.pool)
	if err != nil {
		return domain.Lote{}, err
	}
	if len(lotes) == 0 {
		return domain.Lote{}, nil
	}
	return lotes[0], nil
}

// Vigente compone la respuesta del monitor con el identificador visible. Un lote
// sin id_tasa se publica como T000, igual que en la cartelera calculada.
func (s *Service) Vigente(ctx context.Context) (Monitor, error) {
	lote, err := s.LoteVigente(ctx)
	if err != nil {
		return Monitor{}, err
	}
	id := lote.IDTasa
	if id == "" {
		id = "T000"
	}
	tasas := lote.Tasas
	if tasas == nil {
		tasas = map[string]any{}
	}
	return Monitor{IDTasa: id, Tasas: tasas}, nil
}
