// Package http es la capa de transporte del dominio Tasas. Traduce Fiber a casos
// de uso del servicio y devuelve la forma JSON que consume el panel. No escribe SQL
// ni abre pools: todas las dependencias llegan construidas desde cmd/server.
package http

import (
	"time"

	"glauk/internal/tasas/hoo"
	"glauk/internal/tasas/service"
)

// Deps reúne las piezas compartidas por los handlers.
//
// Service es la única puerta al store y a Evolution. Hoo puede ser nulo cuando el
// origen externo no está configurado: el handler responde un error explícito. Now
// queda disponible para cualquier contrato que necesite un sello reproducible,
// sin que la capa HTTP construya un reloj global por su cuenta.
type Deps struct {
	Service *service.Service
	Hoo     hoo.Borrador
	Now     func() time.Time
}

// NewDeps construye la capa HTTP. Now se inyecta para que las pruebas puedan fijar
// el instante sin mutar el reloj del proceso.
func NewDeps(serv *service.Service, borrador hoo.Borrador, now func() time.Time) *Deps {
	if now == nil {
		now = time.Now
	}
	return &Deps{Service: serv, Hoo: borrador, Now: now}
}
