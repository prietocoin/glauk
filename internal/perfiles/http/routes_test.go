package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	nethttp "net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/perfiles/model"
	"glauk/internal/perfiles/repo"
	"glauk/internal/perfiles/service"
)

type fakeStore struct {
	perfiles  []model.Perfil
	activos   []string
	params    repo.UpsertParams
	eliminado string
}

func (f *fakeStore) Listar(context.Context) ([]model.Perfil, error) { return f.perfiles, nil }
func (f *fakeStore) Activos(context.Context) ([]string, error)      { return f.activos, nil }
func (f *fakeStore) PorNombre(context.Context, string) (model.Perfil, error) {
	return model.Perfil{}, nil
}
func (f *fakeStore) Guardar(_ context.Context, params repo.UpsertParams) (model.Perfil, error) {
	f.params = params
	return model.Perfil{Nombre: params.Nombre, Mostrar: params.Mostrar, Monedas: params.Monedas}, nil
}
func (f *fakeStore) Toggle(_ context.Context, nombre string, activo bool) (model.Perfil, error) {
	return model.Perfil{Nombre: nombre, Mostrar: model.Mostrar{Tasas: activo}}, nil
}
func (f *fakeStore) Delete(_ context.Context, nombre string) error { f.eliminado = nombre; return nil }
func (f *fakeStore) ApagarTodos(context.Context) error             { return nil }
func (f *fakeStore) Memorizar(context.Context) error               { return nil }
func (f *fakeStore) Restaurar(context.Context) error               { return nil }

func appDePrueba(t *testing.T, store *fakeStore) *fiber.App {
	t.Helper()
	app := fiber.New()
	svc := service.New(store, store, nil)
	if err := Register(app, Deps{Perfiles: svc}); err != nil {
		t.Fatal(err)
	}
	return app
}

func pedir(t *testing.T, app *fiber.App, method, ruta, body string) (*nethttp.Response, string) {
	t.Helper()
	var lector io.Reader
	if body != "" {
		lector = strings.NewReader(body)
	}
	req, err := nethttp.NewRequest(method, ruta, lector)
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, string(data)
}

func TestRegisterExigeServicio(t *testing.T) {
	if err := Register(fiber.New(), Deps{}); !errors.Is(err, ErrServicioNoConfigurado) {
		t.Fatalf("err = %v", err)
	}
}

func TestRegisterOchoRutas(t *testing.T) {
	app := appDePrueba(t, &fakeStore{})
	cantidad := 0
	for _, ruta := range app.GetRoutes() {
		if ruta.Method != fiber.MethodHead {
			cantidad++
		}
	}
	if cantidad != 8 {
		t.Fatalf("rutas canónicas = %d, quiero 8", cantidad)
	}
}

func TestRutasLectura(t *testing.T) {
	store := &fakeStore{
		perfiles: []model.Perfil{{Nombre: "GENERAL", Mostrar: model.Mostrar{Tasas: true}, Monedas: model.Monedas{}}},
		activos:  []string{"GENERAL"},
	}
	app := appDePrueba(t, store)
	resp, body := pedir(t, app, nethttp.MethodGet, "/api/v1/perfiles", "")
	if resp.StatusCode != fiber.StatusOK || !strings.HasPrefix(strings.TrimSpace(body), "[") {
		t.Fatalf("listar: status=%d body=%s", resp.StatusCode, body)
	}
	var perfiles []model.Perfil
	if err := json.Unmarshal([]byte(body), &perfiles); err != nil || !perfiles[0].Activo {
		t.Fatalf("listar decode: %v %s", err, body)
	}
	resp, body = pedir(t, app, nethttp.MethodGet, "/api/v1/perfiles/activos", "")
	if resp.StatusCode != fiber.StatusOK || body != `["GENERAL"]` {
		t.Fatalf("activos: status=%d body=%s", resp.StatusCode, body)
	}
}

func TestRutasEscritura(t *testing.T) {
	store := &fakeStore{}
	app := appDePrueba(t, store)
	casos := []struct {
		metodo string
		ruta   string
		body   string
	}{
		{nethttp.MethodPost, "/api/v1/perfiles/guardar", `{"socio":"nuevo","jid":"120@g.us"}`},
		{nethttp.MethodPost, "/api/v1/perfiles/apagar-todos", ""},
		{nethttp.MethodPost, "/api/v1/perfiles/plantilla/memorizar", ""},
		{nethttp.MethodPost, "/api/v1/perfiles/plantilla/restaurar", ""},
		{nethttp.MethodPatch, "/api/v1/perfiles/NUEVO/estado", `{"activo":false}`},
		{nethttp.MethodDelete, "/api/v1/perfiles/NUEVO", ""},
	}
	for _, caso := range casos {
		resp, body := pedir(t, app, caso.metodo, caso.ruta, caso.body)
		if resp.StatusCode != fiber.StatusOK || !strings.Contains(body, `"success":true`) {
			t.Fatalf("%s %s: status=%d body=%s", caso.metodo, caso.ruta, resp.StatusCode, body)
		}
	}
	if store.params.Nombre != "NUEVO" || store.params.IDGrupo != "120@g.us" {
		t.Fatalf("alias no traducidos: %+v", store.params)
	}
}

func TestDeleteProtegido(t *testing.T) {
	app := appDePrueba(t, &fakeStore{})
	resp, body := pedir(t, app, nethttp.MethodDelete, "/api/v1/perfiles/GENERAL", "")
	if resp.StatusCode != fiber.StatusBadRequest || !strings.Contains(body, "ENTIDAD_PROTEGIDA") {
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
}
