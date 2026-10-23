package service

import (
	"context"

	"glauk/internal/perfiles/model"
	"glauk/internal/perfiles/repo"
)

// Lector es la parte de lectura que el servicio necesita. Es una interfaz local
// para poder probar reglas de negocio sin PostgreSQL, sin acoplar el dominio al
// pool ni a pgxmock.
type Lector interface {
	Listar(ctx context.Context) ([]model.Perfil, error)
	Activos(ctx context.Context) ([]string, error)
	PorNombre(ctx context.Context, nombre string) (model.Perfil, error)
}

// Escritor es la parte de escritura del repositorio. El servicio decide defaults
// y protecciones; el repositorio solo ejecuta SQL y aplica el codec canónico.
type Escritor interface {
	Guardar(ctx context.Context, params repo.UpsertParams) (model.Perfil, error)
	Toggle(ctx context.Context, nombre string, activo bool) (model.Perfil, error)
	Delete(ctx context.Context, nombre string) error
	ApagarTodos(ctx context.Context) error
	Memorizar(ctx context.Context) error
	Restaurar(ctx context.Context) error
}

// El repositorio real satisface ambos puertos sin adaptadores.
var (
	_ Lector   = (*repo.Reader)(nil)
	_ Escritor = (*repo.Writer)(nil)
)
