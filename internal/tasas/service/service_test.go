package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"

	"glauk/internal/platform/evolution"
	"glauk/internal/tasas/domain"
	"glauk/internal/tasas/store"
)

const (
	sqlLotes      = `SELECT id_tasa, tasas FROM tasas_glaukov ORDER BY id DESC LIMIT 2`
	sqlSocios     = `SELECT * FROM perfiles_glaukov ORDER BY nombre ASC`
	sqlLockLote   = `SELECT pg_advisory_xact_lock($1)`
	sqlInsertLote = `INSERT INTO tasas_glaukov (id_tasa, tasas) VALUES ($1, $2) RETURNING id_tasa, tasas`
)

var ahoraFijo = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

type remitenteFalso struct {
	mu       sync.Mutex
	fallar   string
	destinos []string
	globos   []string
}

func (r *remitenteFalso) SendMedia(_ context.Context, in evolution.MediaInput) (evolution.Receipt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.destinos = append(r.destinos, in.Number)
	r.globos = append(r.globos, in.Caption)
	if r.fallar != "" && strings.Contains(in.Caption, r.fallar) {
		return evolution.Receipt{}, errors.New("evolution api: estado 401")
	}
	return evolution.Receipt{Status: "SENT", MessageID: "3EB0"}, nil
}

func nuevoMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("no se pudo crear el doble del pool: %v", err)
	}
	t.Cleanup(mock.Close)
	return mock
}

func puntero[T any](v T) *T { return &v }

func TestH02TablaDeVerdad(t *testing.T) {
	casos := []struct {
		nombre  string
		mostrar *bool
		activo  *bool
		quiero  bool
	}{
		{"mostrar ausente", nil, nil, false},
		{"mostrar false apaga aunque activo sea true", puntero(false), puntero(true), false},
		{"mostrar true y activo ausente", puntero(true), nil, true},
		{"mostrar true y activo true", puntero(true), puntero(true), true},
		{"mostrar true y activo false", puntero(true), puntero(false), false},
		{"ambos false", puntero(false), puntero(false), false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			socio := store.Socio{Mostrar: store.Mostrar{Tasas: caso.mostrar}, Activo: caso.activo}
			if obtenido := activoParaTasas(socio); obtenido != caso.quiero {
				t.Fatalf("H-02 devolvió %v, se esperaba %v", obtenido, caso.quiero)
			}
		})
	}
}

func TestPublicarFiltraH02YConservaErroresParciales(t *testing.T) {
	mock := nuevoMock(t)
	mock.ExpectQuery(sqlLotes).WillReturnRows(pgxmock.NewRows([]string{"id_tasa", "tasas"}).
		AddRow(puntero("T060"), []byte(`{"USDT":1,"ARS":1250}`)).
		AddRow(puntero("T059"), []byte(`{"USDT":1,"ARS":1200}`)))
	mock.ExpectQuery(sqlSocios).WillReturnRows(pgxmock.NewRows(
		[]string{"nombre", "id_grupo", "moneda_base", "monedas", "activo", "modo_prueba", "rol", "mostrar"}).
		AddRow("BUENO", "real@g.us", "USDT", []byte(`{}`), nil, nil, "SOCIO", []byte(`{"tasas":true}`)).
		AddRow("MALO", "real@g.us", "USDT", []byte(`{}`), nil, nil, "SOCIO", []byte(`{"tasas":true}`)).
		AddRow("APAGADO", "real@g.us", "USDT", []byte(`{}`), puntero(true), nil, "SOCIO", []byte(`{"tasas":false}`)).
		AddRow("SIN_MOSTRAR", "real@g.us", "USDT", []byte(`{}`), puntero(true), nil, "SOCIO", nil))

	remitente := &remitenteFalso{fallar: "MALO"}
	servicio := newService(mock, remitente, "prueba@g.us", func() time.Time { return ahoraFijo })
	publicacion, err := servicio.Publicar(context.Background(), true)
	if err != nil {
		t.Fatalf("Publicar devolvió error: %v", err)
	}
	if publicacion.IDTasa != "T060" || publicacion.Procesados != 2 || len(publicacion.Detalle) != 2 {
		t.Fatalf("publicación inesperada: %+v", publicacion)
	}
	if publicacion.Detalle[0].Socio != "BUENO" || publicacion.Detalle[0].Status != "ok" {
		t.Fatalf("detalle ordenado o estado incorrecto: %+v", publicacion.Detalle)
	}
	if publicacion.Detalle[1].Socio != "MALO" || publicacion.Detalle[1].Status != "error" {
		t.Fatalf("el error parcial no se conservó: %+v", publicacion.Detalle)
	}
	if len(remitente.destinos) != 2 {
		t.Fatalf("H-02 debía enviar solo 2 carteleras, se enviaron %d", len(remitente.destinos))
	}
	for _, destino := range remitente.destinos {
		if destino != "prueba@g.us" {
			t.Fatalf("el modo prueba debía usar JID_GRUPO_PRUEBA, llegó %q", destino)
		}
	}
	if !strings.Contains(remitente.globos[0], "[PRUEBA]") {
		t.Fatalf("el pie no distingue el modo prueba: %q", remitente.globos[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("consultas pendientes: %v", err)
	}
}

func TestPublicarSinLotes(t *testing.T) {
	mock := nuevoMock(t)
	mock.ExpectQuery(sqlLotes).WillReturnRows(pgxmock.NewRows([]string{"id_tasa", "tasas"}))
	servicio := newService(mock, &remitenteFalso{}, "prueba@g.us", time.Now)
	if _, err := servicio.Publicar(context.Background(), false); !errors.Is(err, ErrSinLotes) {
		t.Fatalf("se esperaba ErrSinLotes, llegó %v", err)
	}
}

func TestGuardarLoteSerializaLecturaYEscritura(t *testing.T) {
	mock := nuevoMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(sqlLockLote).WithArgs(loteLockKey).
		WillReturnResult(pgconn.NewCommandTag("SELECT 1"))
	mock.ExpectQuery(sqlLotes).WillReturnRows(
		pgxmock.NewRows([]string{"id_tasa", "tasas"}).
			AddRow(puntero("T060"), []byte(`{"USDT":1}`)))
	mock.ExpectQuery(sqlInsertLote).
		WithArgs("T061", pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id_tasa", "tasas"}).
			AddRow(puntero("T061"), []byte(`{"COP":4050}`)))
	mock.ExpectCommit()

	servicio := newService(mock, nil, "", func() time.Time { return ahoraFijo })
	tasas := map[string]any{"COP": 4050}
	lote, err := servicio.GuardarLote(context.Background(), tasas)
	if err != nil {
		t.Fatalf("guardar lote: %v", err)
	}
	if lote.IDTasa != "T061" || lote.Tasas["COP"] != 4050.0 {
		t.Fatalf("lote inesperado: %+v", lote)
	}
	if tasas["CREATED_AT"] != ahoraFijo.UTC().Format(selloFecha) {
		t.Fatalf("sello = %v", tasas["CREATED_AT"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("consultas pendientes: %v", err)
	}
}

func TestGuardarLoteReintentaSiOtroEscritorGanaLaCarrera(t *testing.T) {
	mock := nuevoMock(t)

	mock.ExpectBegin()
	mock.ExpectExec(sqlLockLote).WithArgs(loteLockKey).
		WillReturnResult(pgconn.NewCommandTag("SELECT 1"))
	mock.ExpectQuery(sqlLotes).WillReturnRows(
		pgxmock.NewRows([]string{"id_tasa", "tasas"}).
			AddRow(puntero("T060"), []byte(`{"USDT":1}`)))
	mock.ExpectQuery(sqlInsertLote).
		WithArgs("T061", pgxmock.AnyArg()).
		WillReturnError(&pgconn.PgError{Code: "23505"})
	mock.ExpectRollback()

	mock.ExpectBegin()
	mock.ExpectExec(sqlLockLote).WithArgs(loteLockKey).
		WillReturnResult(pgconn.NewCommandTag("SELECT 1"))
	mock.ExpectQuery(sqlLotes).WillReturnRows(
		pgxmock.NewRows([]string{"id_tasa", "tasas"}).
			AddRow(puntero("T061"), []byte(`{"USDT":1}`)))
	mock.ExpectQuery(sqlInsertLote).
		WithArgs("T062", pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id_tasa", "tasas"}).
			AddRow(puntero("T062"), []byte(`{"COP":4050}`)))
	mock.ExpectCommit()

	servicio := newService(mock, nil, "", func() time.Time { return ahoraFijo })
	lote, err := servicio.GuardarLote(context.Background(), map[string]any{"COP": 4050})
	if err != nil {
		t.Fatalf("guardar lote: %v", err)
	}
	if lote.IDTasa != "T062" {
		t.Fatalf("el reintento debía producir T062, llegó %q", lote.IDTasa)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("consultas pendientes: %v", err)
	}
}

func TestDestinoEligeGrupoRealOPrueba(t *testing.T) {
	servicio := newService(nil, nil, "prueba@g.us", time.Now)
	socio := store.Socio{Perfil: domain.Perfil{IDGrupo: "real@g.us"}}
	if destino, _ := servicio.destino(socio, false); destino != "real@g.us" {
		t.Fatalf("modo oficial debía usar el grupo real, llegó %q", destino)
	}
	if destino, _ := servicio.destino(socio, true); destino != "prueba@g.us" {
		t.Fatalf("modo prueba debía usar el JID de pruebas, llegó %q", destino)
	}
	if destino, _ := servicio.destino(store.Socio{}, false); destino != "prueba@g.us" {
		t.Fatalf("un socio sin grupo debía caer en pruebas, llegó %q", destino)
	}
	sinJID := newService(nil, nil, "", time.Now)
	if _, err := sinJID.destino(store.Socio{}, false); err == nil {
		t.Fatal("sin JID de pruebas el envío debe fallar, no inventar destino")
	}
}
