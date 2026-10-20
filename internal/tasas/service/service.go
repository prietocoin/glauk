// Package service orquesta el dominio de Tasas. No conoce Fiber ni escribe SQL:
// recibe el store y un remitente ya construidos, y concentra las reglas que deben
// cumplirse igual desde HTTP, desde un reenvío manual o desde una prueba.
package service

import (
	"time"

	"glauk/internal/platform/config"
	"glauk/internal/tasas/store"
)

// Service es la fachada de casos de uso del dominio Tasas.
//
// El pool nunca se abre aquí: llega como store.Querier desde platform/postgres.
// El remitente también llega inyectado, de modo que las pruebas no toquen la red.
type Service struct {
	pool      store.Querier
	sender    Sender
	jidPrueba string
	now       func() time.Time
}

// New construye el servicio con el reloj del sistema y el JID de pruebas de la
// configuración global. Un JID vacío no se inventa: el envío en modo prueba falla.
func New(pool store.Querier, sender Sender, cfg *config.Config) *Service {
	jid := ""
	if cfg != nil {
		jid = cfg.Tenants.JIDGrupoPrueba
	}
	return newService(pool, sender, jid, time.Now)
}

// newService permite fijar el reloj en pruebas sin exponer estado mutable.
func newService(pool store.Querier, sender Sender, jidPrueba string, now func() time.Time) *Service {
	return &Service{pool: pool, sender: sender, jidPrueba: jidPrueba, now: now}
}
