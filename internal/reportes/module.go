package reportes

import "context"

// Module agrupa el servicio puro con sus colaboradores de proceso. No importa
// Fiber ni adaptadores: cmd/server construye los puertos y llama NewModule.
type Module struct {
	Service     *Service
	Idempotency *IdempotencyStore
	RateLimiter *RateLimiter
}

// NewModule ancla el servicio y crea el estado en memoria de idempotencia y
// rate limiting. Si el servicio es nil, devuelve un módulo vacío para que el
// cableado falle explícitamente al registrar rutas, no en el primer request.
func NewModule(service *Service) *Module {
	if service == nil {
		return &Module{}
	}
	return &Module{
		Service:     service,
		Idempotency: NewIdempotencyStore(service.cfg.IdempotencyTTL),
		RateLimiter: NewRateLimiter(defaultRateLimitPerMinute),
	}
}

// Valido indica si el módulo puede atender tráfico.
func (m *Module) Valido() bool {
	return m != nil && m.Service != nil
}

// Filtros delega en el servicio para que el handler no acceda a campos
// internos ni conozca la firma del proveedor de movimientos.
func (m *Module) Filtros(ctx context.Context, rol string) (Filtros, error) {
	return m.Service.Filtros(ctx, rol)
}
