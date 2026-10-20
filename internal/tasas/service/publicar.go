package service

import (
	"context"
	"errors"

	"glauk/internal/tasas/calculator"
	"glauk/internal/tasas/domain"
	"glauk/internal/tasas/store"
)

// ErrSinLotes es un fallo de entrada, no del servidor: el handler lo traduce a 400
// con el mismo texto que publicaba el servicio Node.
var ErrSinLotes = errors.New("No hay lotes guardados en tasas_glaukov para publicar.")

// Publicacion es la respuesta común de publicar, disparar un lote y reenviar.
type Publicacion struct {
	IDTasa     string      `json:"id_tasa"`
	Procesados int         `json:"procesados"`
	Detalle    []Resultado `json:"detalle"`
}

// Publicar envía el lote vigente a todos los socios activos según H-02. Los errores
// parciales no abortan el resto: quedan en Detalle y no convierten la respuesta en 500.
func (s *Service) Publicar(ctx context.Context, modoPruebaGlobal bool) (Publicacion, error) {
	lotes, err := store.RecentLotes(ctx, s.pool)
	if err != nil {
		return Publicacion{}, err
	}
	if len(lotes) == 0 {
		return Publicacion{}, ErrSinLotes
	}
	actual, anterior := loteVigente(lotes)
	socios, err := store.AllSocios(ctx, s.pool)
	if err != nil {
		return Publicacion{}, err
	}
	return s.publicarCarteleras(ctx, actual, anterior, activosDe(socios), modoPruebaGlobal), nil
}

// publicarCarteleras comparte el bucle de envío entre publicación y reenvío.
func (s *Service) publicarCarteleras(
	ctx context.Context,
	actual, anterior domain.Lote,
	socios []store.Socio,
	modoPrueba bool,
) Publicacion {
	detalle := s.enParalelo(ctx, socios, func(ctx context.Context, socio store.Socio) Resultado {
		cartelera := calculator.Build(socio.Perfil, actual, anterior, s.now())
		if _, err := s.enviarCartelera(ctx, socio, cartelera, modoPrueba); err != nil {
			return Resultado{Socio: socio.Nombre, Status: "error", Error: err.Error()}
		}
		return Resultado{Socio: socio.Nombre, Status: "ok"}
	})
	return Publicacion{IDTasa: actual.IDTasa, Procesados: len(detalle), Detalle: detalle}
}

// activosDe aplica H-02 antes de publicar. mostrar.tasas debe ser exactamente true;
// la columna activo, cuando existe, solo puede apagar: un true nunca enciende a un
// socio que Perfiles apagó.
func activosDe(socios []store.Socio) []store.Socio {
	activos := make([]store.Socio, 0, len(socios))
	for _, socio := range socios {
		if !activoParaTasas(socio) {
			continue
		}
		activos = append(activos, socio)
	}
	return activos
}

// activoParaTasas concentra la tabla de verdad de H-02 en una sola función testeable.
func activoParaTasas(socio store.Socio) bool {
	if socio.Mostrar.Tasas == nil || !*socio.Mostrar.Tasas {
		return false
	}
	return socio.Activo == nil || *socio.Activo
}
