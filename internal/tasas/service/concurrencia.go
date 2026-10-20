package service

import (
	"context"
	"sort"
	"sync"
	"time"

	"glauk/internal/tasas/store"
)

// workers acota los envíos simultáneos. Evolution atiende una sola instancia de
// WhatsApp: lanzar todos los socios a la vez cambia una publicación correcta por
// una tanda de errores de red.
const workers = 5

// timeoutSocio limita renderizar y enviar una cartelera. Sin este plazo, un socio
// lento retendría su worker y retrasaría al resto de la publicación.
const timeoutSocio = 30 * time.Second

// Resultado es la fila del detalle que consume el panel para cada socio.
type Resultado struct {
	Socio  string `json:"socio"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// enParalelo aplica el trabajo con cinco workers y un plazo independiente por
// socio. La salida se ordena por nombre para que el panel no cambie de orden entre
// publicaciones: el orden de finalización del pool es impredecible.
func (s *Service) enParalelo(
	ctx context.Context,
	socios []store.Socio,
	trabajo func(context.Context, store.Socio) Resultado,
) []Resultado {
	salida := make([]Resultado, len(socios))
	limite := make(chan struct{}, workers)
	var grupo sync.WaitGroup
	for i, socio := range socios {
		grupo.Add(1)
		go func(i int, socio store.Socio) {
			defer grupo.Done()
			limite <- struct{}{}
			defer func() { <-limite }()
			acotado, cancelar := context.WithTimeout(ctx, timeoutSocio)
			defer cancelar()
			salida[i] = trabajo(acotado, socio)
		}(i, socio)
	}
	grupo.Wait()
	sort.Slice(salida, func(i, j int) bool { return salida[i].Socio < salida[j].Socio })
	return salida
}
