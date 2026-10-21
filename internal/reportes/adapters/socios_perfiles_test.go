package adapters

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"glauk/internal/perfiles/model"
	"glauk/internal/reportes"
)

type fakeSocioReader struct {
	perfiles []model.Perfil
	err      error
	porNom   model.Perfil
	porErr   error
}

func (f fakeSocioReader) Listar(context.Context) ([]model.Perfil, error) {
	return f.perfiles, f.err
}

func (f fakeSocioReader) PorNombre(context.Context, string) (model.Perfil, error) {
	return f.porNom, f.porErr
}

func TestSociosPerfilesProyectaYFiltra(t *testing.T) {
	reader := fakeSocioReader{perfiles: []model.Perfil{
		{Nombre: "GENERAL", Rol: "ADMIN", IDGrupo: "1@g.us", Activo: true, MonedaBase: "USD"},
		{Nombre: "FUNDABLOCK", Rol: "SOCIO", IDGrupo: "2@g.us", Activo: false, MonedaBase: "USDT"},
	}}
	adapter := NewSociosPerfiles(reader)

	todos, err := adapter.Socios(context.Background(), "")
	if err != nil || len(todos) != 2 {
		t.Fatalf("Socios() = %d, %v", len(todos), err)
	}
	if todos[0].Nombre != "GENERAL" || !todos[0].Activo || todos[0].Moneda != "USD" {
		t.Fatalf("proyección inesperada: %+v", todos[0])
	}

	socios, err := adapter.Socios(context.Background(), " socio ")
	if err != nil || len(socios) != 1 || socios[0].Nombre != "FUNDABLOCK" {
		t.Fatalf("filtro por rol = %+v, %v", socios, err)
	}
}

func TestSociosPerfilesNoEncontrado(t *testing.T) {
	adapter := NewSociosPerfiles(fakeSocioReader{porErr: pgx.ErrNoRows})
	_, err := adapter.SocioPorNombre(context.Background(), "NADIE")
	if !errors.Is(err, reportes.ErrSocioNoEncontrado) {
		t.Fatalf("err = %v", err)
	}

	adapter = NewSociosPerfiles(fakeSocioReader{porErr: errors.New("db caída")})
	if _, err := adapter.SocioPorNombre(context.Background(), "NADIE"); err == nil ||
		errors.Is(err, reportes.ErrSocioNoEncontrado) {
		t.Fatalf("un error de infraestructura no debe convertirse en 404: %v", err)
	}
}

func TestSociosPerfilesSocioPorNombre(t *testing.T) {
	adapter := NewSociosPerfiles(fakeSocioReader{porNom: model.Perfil{
		Nombre: "FUNDABLOCK", Rol: "SOCIO", IDGrupo: "2@g.us",
		Activo: true, MonedaBase: "USDT",
	}})
	socio, err := adapter.SocioPorNombre(context.Background(), "FUNDABLOCK")
	if err != nil || socio.Nombre != "FUNDABLOCK" || socio.IDGrupo != "2@g.us" ||
		!socio.Activo || socio.Moneda != "USDT" {
		t.Fatalf("socio = %+v, err = %v", socio, err)
	}
}
