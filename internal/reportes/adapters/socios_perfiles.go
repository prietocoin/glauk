// Package adapters conecta el dominio puro de reportes con el resto del
// monólito. Es el único lugar del módulo que conoce perfiles, comprobantes,
// Evolution y pgx.
package adapters

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"glauk/internal/perfiles/model"
	"glauk/internal/reportes"
)

// SocioReader es el puerto de lectura que reportes necesita del directorio.
// Lo satisface *repo.Reader y cualquier doble de prueba estructural.
type SocioReader interface {
	Listar(ctx context.Context) ([]model.Perfil, error)
	PorNombre(ctx context.Context, nombre string) (model.Perfil, error)
}

// SociosPerfiles adapta el directorio a reportes.SociosProvider.
type SociosPerfiles struct {
	reader SocioReader
}

// NewSociosPerfiles construye el adaptador con el lector ya inyectado.
func NewSociosPerfiles(reader SocioReader) *SociosPerfiles {
	return &SociosPerfiles{reader: reader}
}

// SocioPorNombre traduce el no encontrado de pgx al error público del dominio.
func (a *SociosPerfiles) SocioPorNombre(ctx context.Context, nombre string) (reportes.Socio, error) {
	perfil, err := a.reader.PorNombre(ctx, nombre)
	if errors.Is(err, pgx.ErrNoRows) {
		return reportes.Socio{}, reportes.ErrSocioNoEncontrado
	}
	if err != nil {
		return reportes.Socio{}, err
	}
	return socioDe(perfil), nil
}

// Socios devuelve el directorio proyectado y filtra por rol cuando se pide.
func (a *SociosPerfiles) Socios(ctx context.Context, rol string) ([]reportes.Socio, error) {
	perfiles, err := a.reader.Listar(ctx)
	if err != nil {
		return nil, err
	}
	rol = strings.TrimSpace(rol)
	out := make([]reportes.Socio, 0, len(perfiles))
	for _, perfil := range perfiles {
		if rol != "" && !strings.EqualFold(strings.TrimSpace(perfil.Rol), rol) {
			continue
		}
		out = append(out, socioDe(perfil))
	}
	return out, nil
}

func socioDe(perfil model.Perfil) reportes.Socio {
	return reportes.Socio{
		Nombre:  perfil.Nombre,
		Rol:     perfil.Rol,
		IDGrupo: perfil.IDGrupo,
		Activo:  perfil.Activo,
		Moneda:  perfil.MonedaBase,
	}
}
