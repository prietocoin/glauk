package service

import (
	"context"
	"errors"
	"strings"

	"glauk/internal/tasas/domain"
	"glauk/internal/tasas/store"
)

// ErrTasasInvalidas rechaza un lote sin tasas antes de llegar a PostgreSQL.
var ErrTasasInvalidas = errors.New("Estructura de tasas inválida.")

// selloFecha reproduce new Date().toISOString(): milisegundos y zona Z.
const selloFecha = "2006-01-02T15:04:05.000Z"

// GuardarLote inserta un lote nuevo sin publicarlo. El identificador se deriva del
// último lote y el sello se completa en UTC cuando el panel no lo envía.
func (s *Service) GuardarLote(ctx context.Context, tasas map[string]any) (domain.Lote, error) {
	if len(tasas) == 0 {
		return domain.Lote{}, ErrTasasInvalidas
	}
	lotes, err := store.RecentLotes(ctx, s.pool)
	if err != nil {
		return domain.Lote{}, err
	}
	ultimo := ""
	if len(lotes) > 0 {
		ultimo = lotes[0].IDTasa
	}
	if sello, ok := tasas["CREATED_AT"].(string); !ok || sello == "" {
		tasas["CREATED_AT"] = s.now().UTC().Format(selloFecha)
	}
	return store.InsertLote(ctx, s.pool, store.NextLoteID(ultimo), tasas)
}

// EliminarLote borra un lote concreto e informa si existía.
func (s *Service) EliminarLote(ctx context.Context, idTasa string) (bool, error) {
	idTasa = strings.TrimSpace(idTasa)
	if idTasa == "" {
		return false, ErrLoteInvalido
	}
	return store.DeleteLote(ctx, s.pool, idTasa)
}
