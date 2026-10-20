// Package service aplica las reglas del dominio Perfiles sobre el repositorio.
//
// Es el único lugar que decide defaults, protege las entidades matriz y aplica
// H-02. No escribe SQL ni conoce Fiber: ambas cosas viven en repo y http.
package service

import (
	"context"
	"log/slog"

	"glauk/internal/perfiles/model"
)

// Service es la fachada de casos de uso del directorio.
type Service struct {
	lector   Lector
	escritor Escritor
	log      *slog.Logger
}

// New construye el servicio con dependencias explícitas. Un logger nulo cae al
// logger global para no perder los WARN de H-01 y H-03.
func New(lector Lector, escritor Escritor, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{lector: lector, escritor: escritor, log: log}
}

// Listar devuelve el directorio con activo derivado de H-02.
func (s *Service) Listar(ctx context.Context) ([]model.Perfil, error) {
	perfiles, err := s.lector.Listar(ctx)
	if err != nil {
		return nil, err
	}
	for i := range perfiles {
		perfiles[i].Activo = ActividadDePerfil(perfiles[i])
	}
	return perfiles, nil
}

// Activos devuelve los nombres que el motor de Tasas puede publicar.
func (s *Service) Activos(ctx context.Context) ([]string, error) {
	return s.lector.Activos(ctx)
}

// Guardar valida el nombre, emite los WARN del codec y delega el upsert atómico.
func (s *Service) Guardar(ctx context.Context, req model.GuardarRequest) (model.Perfil, error) {
	nombre := NormalizarNombre(req.Nombre)
	if nombre == "" {
		return model.Perfil{}, ErrNombreRequerido
	}
	advertirMonedas(nombre, req.Monedas, s.log)
	perfil, err := s.escritor.Guardar(ctx, buildParams(nombre, req, s.log))
	if err != nil {
		return model.Perfil{}, err
	}
	perfil.Activo = ActividadDePerfil(perfil)
	return perfil, nil
}

// Toggle escribe mostrar.tasas. GENERAL no puede apagarse: es la matriz del panel.
func (s *Service) Toggle(ctx context.Context, nombre string, activo bool) (model.Perfil, error) {
	nombre = NormalizarNombre(nombre)
	if nombre == "" {
		return model.Perfil{}, ErrNombreRequerido
	}
	if nombre == "GENERAL" && !activo {
		s.log.Warn("se ignoró el apagado de la entidad matriz", "socio", nombre)
		activo = true
	}
	perfil, err := s.escritor.Toggle(ctx, nombre, activo)
	if err != nil {
		return model.Perfil{}, err
	}
	perfil.Activo = ActividadDePerfil(perfil)
	return perfil, nil
}

// Eliminar bloquea las matrices GENERAL y FUNDDA antes de tocar la base.
func (s *Service) Eliminar(ctx context.Context, nombre string) error {
	nombre = NormalizarNombre(nombre)
	if nombre == "" {
		return ErrNombreRequerido
	}
	if nombre == "GENERAL" || nombre == "FUNDDA" {
		return ErrEntidadProtegida
	}
	return s.escritor.Delete(ctx, nombre)
}
