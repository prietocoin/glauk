package service

import (
	"context"
	"errors"
	"strings"

	"glauk/internal/tasas/store"
)

// ErrLoteInvalido señala que falta el identificador del lote a publicar.
var ErrLoteInvalido = errors.New("Debe proporcionar un ID de lote válido.")

// PublicarLote publica un lote concreto, no necesariamente el vigente. Conserva la
// compatibilidad con /api/tasas/publicar del servicio anterior.
func (s *Service) PublicarLote(ctx context.Context, idTasa string, modoPruebaGlobal bool) (Publicacion, error) {
	idTasa = strings.TrimSpace(idTasa)
	if idTasa == "" {
		return Publicacion{}, ErrLoteInvalido
	}
	actual, err := store.LoteByID(ctx, s.pool, idTasa)
	if err != nil {
		return Publicacion{}, err
	}
	lotes, err := store.RecentLotes(ctx, s.pool)
	if err != nil {
		return Publicacion{}, err
	}
	socios, err := store.AllSocios(ctx, s.pool)
	if err != nil {
		return Publicacion{}, err
	}
	anterior := anteriorDe(lotes, actual)
	return s.publicarCarteleras(ctx, actual, anterior, activosDe(socios), modoPruebaGlobal), nil
}
