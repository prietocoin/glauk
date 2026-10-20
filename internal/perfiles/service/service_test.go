package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"glauk/internal/perfiles/model"
	"glauk/internal/perfiles/repo"
)

type fakeLector struct {
	perfiles []model.Perfil
	activos  []string
}

func (f *fakeLector) Listar(context.Context) ([]model.Perfil, error) { return f.perfiles, nil }
func (f *fakeLector) Activos(context.Context) ([]string, error)      { return f.activos, nil }
func (f *fakeLector) PorNombre(context.Context, string) (model.Perfil, error) {
	return model.Perfil{}, nil
}

type fakeEscritor struct {
	params       repo.UpsertParams
	guardar      model.Perfil
	togglePerfil model.Perfil
	toggleActivo bool
	eliminado    string
	apagado      bool
	memorizado   bool
	restaurado   bool
}

func (f *fakeEscritor) Guardar(_ context.Context, params repo.UpsertParams) (model.Perfil, error) {
	f.params = params
	return f.guardar, nil
}
func (f *fakeEscritor) Toggle(_ context.Context, _ string, activo bool) (model.Perfil, error) {
	f.toggleActivo = activo
	return f.togglePerfil, nil
}
func (f *fakeEscritor) Delete(_ context.Context, nombre string) error {
	f.eliminado = nombre
	return nil
}
func (f *fakeEscritor) ApagarTodos(context.Context) error { f.apagado = true; return nil }
func (f *fakeEscritor) Memorizar(context.Context) error   { f.memorizado = true; return nil }
func (f *fakeEscritor) Restaurar(context.Context) error   { f.restaurado = true; return nil }

func loggerDePrueba(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestActividadH02(t *testing.T) {
	verdadero, falso := true, false
	casos := []struct {
		mostrar model.Mostrar
		columna *bool
		quiero  bool
	}{
		{model.Mostrar{Tasas: true}, nil, true},
		{model.Mostrar{Tasas: false}, nil, false},
		{model.Mostrar{Tasas: true}, &falso, false},
		{model.Mostrar{Tasas: false}, &verdadero, false},
		{model.Mostrar{Tasas: true}, &verdadero, true},
	}
	for _, caso := range casos {
		if got := Actividad(caso.mostrar, caso.columna); got != caso.quiero {
			t.Fatalf("Actividad(%+v,%v)=%v, quiero %v", caso.mostrar, caso.columna, got, caso.quiero)
		}
	}
}

func TestListarDerivaActivo(t *testing.T) {
	lector := &fakeLector{perfiles: []model.Perfil{
		{Nombre: "GENERAL", Mostrar: model.Mostrar{Tasas: true}},
		{Nombre: "APAGADO", Mostrar: model.Mostrar{Tasas: false}},
	}}
	svc := New(lector, &fakeEscritor{}, nil)
	perfiles, err := svc.Listar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !perfiles[0].Activo || perfiles[1].Activo {
		t.Fatalf("activo derivado incorrecto: %+v", perfiles)
	}
}

func TestGuardarDefaultsYSinJID(t *testing.T) {
	var buf bytes.Buffer
	escritor := &fakeEscritor{guardar: model.Perfil{Nombre: "NUEVO", Mostrar: model.MostrarVisible}}
	svc := New(&fakeLector{}, escritor, loggerDePrueba(&buf))
	perfil, err := svc.Guardar(context.Background(), model.GuardarRequest{Nombre: " nuevo "})
	if err != nil {
		t.Fatal(err)
	}
	if !perfil.Activo || escritor.params.IDGrupo != "" || escritor.params.Nombre != "NUEVO" {
		t.Fatalf("params = %+v", escritor.params)
	}
	if escritor.params.Rol != "SOCIO" || escritor.params.MonedaBase != "USDT" || escritor.params.SaldoInicial != "0" {
		t.Fatalf("defaults = %+v", escritor.params)
	}
	if escritor.params.Monedas == nil || !escritor.params.Mostrar.Tasas {
		t.Fatalf("monedas/mostrar = %+v", escritor.params)
	}
	log := buf.String()
	if !strings.Contains(log, "id_grupo vacío") || strings.Contains(log, "_"+"JID") {
		t.Fatalf("log H-03 = %q", log)
	}
}

func TestGuardarConflictoEmiteWarn(t *testing.T) {
	var buf bytes.Buffer
	escritor := &fakeEscritor{guardar: model.Perfil{Nombre: "FUNDABLOCK", Mostrar: model.MostrarVisible}}
	svc := New(&fakeLector{}, escritor, loggerDePrueba(&buf))
	var req model.GuardarRequest
	raw := `{"nombre":"FUNDABLOCK","monedas":{"ars":{"porcentaje":{"deposito":5},"deposito":3}}}`
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Guardar(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if escritor.params.Monedas["ars"].Deposito != 5 {
		t.Fatalf("monedas = %+v", escritor.params.Monedas)
	}
	log := buf.String()
	if !strings.Contains(log, "porcentaje gana") || !strings.Contains(log, "FUNDABLOCK") || !strings.Contains(log, "ARS") {
		t.Fatalf("WARN H-01 = %q", log)
	}
}

func TestEliminarProtegidas(t *testing.T) {
	escritor := &fakeEscritor{}
	svc := New(&fakeLector{}, escritor, nil)
	for _, nombre := range []string{"GENERAL", " fundda "} {
		if err := svc.Eliminar(context.Background(), nombre); !errors.Is(err, ErrEntidadProtegida) {
			t.Fatalf("%s: err=%v", nombre, err)
		}
	}
	if escritor.eliminado != "" {
		t.Fatalf("el repositorio fue llamado: %q", escritor.eliminado)
	}
	if err := svc.Eliminar(context.Background(), "SOCIO"); err != nil || escritor.eliminado != "SOCIO" {
		t.Fatalf("socio normal: err=%v eliminado=%q", err, escritor.eliminado)
	}
}

func TestToggleGeneralNoSeApaga(t *testing.T) {
	var buf bytes.Buffer
	escritor := &fakeEscritor{togglePerfil: model.Perfil{Mostrar: model.MostrarVisible}}
	svc := New(&fakeLector{}, escritor, loggerDePrueba(&buf))
	if _, err := svc.Toggle(context.Background(), "general", false); err != nil {
		t.Fatal(err)
	}
	if !escritor.toggleActivo || !strings.Contains(buf.String(), "entidad matriz") {
		t.Fatalf("GENERAL se apagó: activo=%v log=%q", escritor.toggleActivo, buf.String())
	}
}

func TestPlantillaDelega(t *testing.T) {
	escritor := &fakeEscritor{}
	svc := New(&fakeLector{}, escritor, nil)
	if err := svc.ApagarTodos(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := svc.Memorizar(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := svc.Restaurar(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !escritor.apagado || !escritor.memorizado || !escritor.restaurado {
		t.Fatalf("plantilla no delegada: %+v", escritor)
	}
}
