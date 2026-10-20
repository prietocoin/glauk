package service

import (
	"context"

	"glauk/internal/tasas/calculator"
	"glauk/internal/tasas/domain"
	"glauk/internal/tasas/store"
)

// CarteleraDe calcula la vista de un socio con el lote vigente y el anterior.
// Es la única definición de "lote vigente" para /calcular, /render y /disparar.
func (s *Service) CarteleraDe(ctx context.Context, nombre string) (domain.Cartelera, error) {
	socio, err := store.SocioByName(ctx, s.pool, nombre)
	if err != nil {
		return domain.Cartelera{}, err
	}
	return s.carteleraDeSocio(ctx, socio)
}

// carteleraDeSocio evita volver a leer el perfil cuando el llamador ya lo tiene.
func (s *Service) carteleraDeSocio(ctx context.Context, socio store.Socio) (domain.Cartelera, error) {
	lotes, err := store.RecentLotes(ctx, s.pool)
	if err != nil {
		return domain.Cartelera{}, err
	}
	actual, anterior := loteVigente(lotes)
	return calculator.Build(socio.Perfil, actual, anterior, s.now()), nil
}

// loteVigente devuelve el lote actual y el anterior. Con un solo lote, el anterior
// es el mismo: es la regla del servicio original y evita una tendencia sin base.
func loteVigente(lotes []domain.Lote) (domain.Lote, domain.Lote) {
	if len(lotes) == 0 {
		return domain.Lote{}, domain.Lote{}
	}
	if len(lotes) == 1 {
		return lotes[0], lotes[0]
	}
	return lotes[0], lotes[1]
}

// anteriorDe busca el lote que precede al publicado cuando se elige por ID.
// Si no está entre los recientes, el cálculo compara el lote consigo mismo.
func anteriorDe(lotes []domain.Lote, actual domain.Lote) domain.Lote {
	for i, lote := range lotes {
		if lote.IDTasa != actual.IDTasa {
			continue
		}
		if i+1 < len(lotes) {
			return lotes[i+1]
		}
		break
	}
	return actual
}
