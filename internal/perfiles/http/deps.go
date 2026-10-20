// Package http expone los casos de uso de Perfiles por Fiber.
//
// Solo monta las ocho rutas canónicas de §5.4.1. Los alias históricos se
// registran en cmd/server con el mismo Deps, de modo que este paquete no duplica
// contratos ni crea un segundo escritor.
package http

import "glauk/internal/perfiles/service"

// BasePath es la superficie canónica de §5.4.1. cmd/server reutiliza esta
// constante para los alias históricos, sin registrar un segundo conjunto de
// handlers ni un segundo escritor de perfiles_glaukov.
const BasePath = "/api/v1/perfiles"

// Deps reúne las dependencias de la capa HTTP. No contiene pools ni lectores de
// entorno: eso pertenece a platform y a cmd/server.
type Deps struct {
	Perfiles *service.Service
}

// NewDeps construye las dependencias desde el servicio ya cableado.
func NewDeps(perfiles *service.Service) Deps {
	return Deps{Perfiles: perfiles}
}

// Valida evita registrar handlers que fallarían en el primer request.
func (d Deps) Valida() error {
	if d.Perfiles == nil {
		return ErrServicioNoConfigurado
	}
	return nil
}
