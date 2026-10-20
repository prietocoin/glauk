package service

import (
	"context"

	"glauk/internal/platform/evolution"
	"glauk/internal/tasas/calculator"
	"glauk/internal/tasas/store"
)

// Disparo conserva el nombre canónico del socio junto a la respuesta de Evolution.
type Disparo struct {
	Socio     string
	Respuesta evolution.Receipt
}

// Disparar envía la cartelera de un solo socio. El modo del cuerpo manda; si no
// viene, se usa el que el socio tiene guardado, igual que el servicio anterior.
func (s *Service) Disparar(ctx context.Context, nombre string, modoPrueba *bool) (Disparo, error) {
	socio, err := store.SocioByName(ctx, s.pool, nombre)
	if err != nil {
		return Disparo{}, err
	}
	cartelera, err := s.carteleraDeSocio(ctx, socio)
	if err != nil {
		return Disparo{}, err
	}
	respuesta, err := s.enviarCartelera(ctx, socio, cartelera, modoPruebaDe(modoPrueba, socio))
	if err != nil {
		return Disparo{}, err
	}
	return Disparo{Socio: socio.Nombre, Respuesta: respuesta}, nil
}

// Reenviar publica de nuevo a un socio concreto o a todos los activos. Con nombre
// vacío aplica H-02; con nombre explícito respeta la decisión manual del operador.
func (s *Service) Reenviar(ctx context.Context, nombre string, modoPrueba bool) (Publicacion, error) {
	if nombre == "" {
		return s.Publicar(ctx, modoPrueba)
	}
	socio, err := store.SocioByName(ctx, s.pool, nombre)
	if err != nil {
		return Publicacion{}, err
	}
	lotes, err := store.RecentLotes(ctx, s.pool)
	if err != nil {
		return Publicacion{}, err
	}
	actual, anterior := loteVigente(lotes)
	cartelera := calculator.Build(socio.Perfil, actual, anterior, s.now())
	resultado := Resultado{Socio: socio.Nombre, Status: "ok"}
	if _, err := s.enviarCartelera(ctx, socio, cartelera, modoPrueba); err != nil {
		resultado.Status = "error"
		resultado.Error = err.Error()
	}
	return Publicacion{IDTasa: actual.IDTasa, Procesados: 1, Detalle: []Resultado{resultado}}, nil
}

// modoPruebaDe resuelve el modo de un disparo individual.
func modoPruebaDe(enviado *bool, socio store.Socio) bool {
	if enviado != nil {
		return *enviado
	}
	return socio.ModoPrueba != nil && *socio.ModoPrueba
}
