// Package http expone los casos de uso de Comprobantes por Fiber.
//
// Solo registra las siete rutas canónicas de §5.5.1 y las tres de la consola
// de §5.8. Los alias históricos pertenecen a cmd/server, que reutiliza este
// mismo Deps para no duplicar reglas.
package http

import (
	"context"

	"glauk/internal/comprobantes/domain"
	"glauk/internal/comprobantes/dto"
	"glauk/internal/platform/config"
)

// BasePath es la superficie canónica de Comprobantes.
const BasePath = "/api/v1/comprobantes"

// Service es el subconjunto del servicio que consume la capa HTTP.
type Service interface {
	List(ctx context.Context, filters domain.ListFilters) ([]domain.Comprobante, error)
	Latest(ctx context.Context) (domain.LoteTasa, error)
	History(ctx context.Context) ([]domain.LoteTasa, error)
	Liquidar(ctx context.Context, input dto.LiquidarInput) (domain.Liquidacion, error)
	Update(ctx context.Context, hash string, input dto.UpdateInput) (domain.Liquidacion, error)
	Delete(ctx context.Context, hash string) error
	Releer(ctx context.Context, hash string) error
	Queue(ctx context.Context) ([]domain.QueueItem, error)
	UpdateQueue(ctx context.Context, hash string, input dto.UpdateInput) (domain.QueueUpdate, error)
	DeleteQueue(ctx context.Context, hash string) error
}

// Deps reúne las dependencias de los handlers.
type Deps struct {
	Service  Service
	AdminKey string
}

// NewDeps cablea el servicio y toma la clave desde config.Admin.APIKey.
func NewDeps(serv Service, cfg *config.Config) Deps {
	key := ""
	if cfg != nil {
		key = cfg.Admin.APIKey
	}
	return Deps{Service: serv, AdminKey: key}
}

// Valida evita registrar handlers que fallarían en el primer request.
func (d Deps) Valida() error {
	if d.Service == nil {
		return ErrServiceNotConfigured
	}
	return nil
}
