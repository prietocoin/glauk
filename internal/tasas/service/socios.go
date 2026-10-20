package service

import (
	"context"

	"glauk/internal/tasas/domain"
	"glauk/internal/tasas/store"
)

// Socios devuelve la lista del panel con los defaults que espera la interfaz:
// monedas nunca es null y un rol vacío se publica como SOCIO.
func (s *Service) Socios(ctx context.Context) ([]store.Socio, error) {
	socios, err := store.AllSocios(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	for i := range socios {
		normalizarSocio(&socios[i])
	}
	return socios, nil
}

// GuardarSocio aplica cambios parciales y devuelve la fila resultante. Un nombre
// inexistente conserva store.ErrNotFound para que HTTP responda 404.
func (s *Service) GuardarSocio(ctx context.Context, nombre string, cambios store.SocioUpdate) (store.Socio, error) {
	socio, err := store.UpdateSocio(ctx, s.pool, nombre, cambios)
	if err != nil {
		return store.Socio{}, err
	}
	normalizarSocio(&socio)
	return socio, nil
}

// normalizarSocio aplica los defaults de presentación sin tocar la actividad H-02.
func normalizarSocio(socio *store.Socio) {
	if socio.Rol == "" {
		socio.Rol = "SOCIO"
	}
	if socio.Monedas == nil {
		socio.Monedas = map[string]*domain.Moneda{}
	}
}
