package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"

	"glauk/internal/platform/config"
	"glauk/internal/platform/evolution"
	"glauk/internal/tasas/hoo"
	"glauk/internal/tasas/service"
)

var columnasSocio = []string{"nombre", "id_grupo", "moneda_base", "monedas", "activo", "modo_prueba", "rol", "mostrar"}

type baseFalsa struct{}

func (baseFalsa) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	switch {
	case strings.Contains(sql, "ORDER BY id DESC"):
		return pgxmock.NewRows([]string{"id_tasa", "tasas"}).
			AddRow(puntero("T060"), []byte(`{"USDT":1,"ARS":1250}`)).
			AddRow(puntero("T059"), []byte(`{"USDT":1,"ARS":1200}`)).Kind(), nil
	case strings.Contains(sql, "WHERE LOWER(nombre)"), strings.HasPrefix(sql, "UPDATE"):
		return pgxmock.NewRows(columnasSocio).AddRow("ANA", "real@g.us", "USDT", []byte(`{}`), nil, nil, "SOCIO", []byte(`{"tasas":true}`)).Kind(), nil
	case strings.Contains(sql, "ORDER BY nombre"):
		return pgxmock.NewRows(columnasSocio).AddRow("ANA", "real@g.us", "USDT", []byte(`{}`), nil, nil, "SOCIO", []byte(`{"tasas":true}`)).Kind(), nil
	case strings.Contains(sql, "LIMIT 0"):
		return pgxmock.NewRows(columnasSocio).Kind(), nil
	case strings.Contains(sql, "INSERT INTO tasas_glaukov"):
		return pgxmock.NewRows([]string{"id_tasa", "tasas"}).AddRow(puntero("T061"), []byte(`{"USDT":1}`)).Kind(), nil
	case strings.Contains(sql, "INSERT INTO notificaciones_tasas"):
		return pgxmock.NewRows([]string{"id_tasa"}).AddRow("T060").Kind(), nil
	default:
		return pgxmock.NewRows([]string{"id_tasa", "tasas"}).Kind(), nil
	}
}

func (baseFalsa) QueryRow(context.Context, string, ...any) pgx.Row { return nil }

func (baseFalsa) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("DELETE 1"), nil
}

// txFalsa permite que el smoke test de guardar-lote recorra la transacción con
// advisory lock sin una base real. Solo se usan Exec, Query, QueryRow, Commit y
// Rollback; el resto del contrato pgx.Tx queda embebido.
type txFalsa struct {
	pgx.Tx
}

func (txFalsa) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return (baseFalsa{}).Exec(ctx, sql, args...)
}

func (txFalsa) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return (baseFalsa{}).Query(ctx, sql, args...)
}

func (txFalsa) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return (baseFalsa{}).QueryRow(ctx, sql, args...)
}

func (txFalsa) Commit(context.Context) error   { return nil }
func (txFalsa) Rollback(context.Context) error { return nil }

func (baseFalsa) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return txFalsa{}, nil
}

type borradorFalso struct{}

func (borradorFalso) Obtener(context.Context) (hoo.Cambios, error) {
	return hoo.Cambios{ID: "borrador-1", USD: puntero(1.0)}, nil
}

func puntero[T any](v T) *T { return &v }

func nuevaApp(t *testing.T) *fiber.App {
	t.Helper()
	cfg := &config.Config{Tenants: config.TenantConfig{JIDGrupoPrueba: "prueba@g.us"}}
	remitente := evolution.New(config.EvolutionConfig{})
	servicio := service.New(baseFalsa{}, remitente, cfg)
	app := fiber.New()
	Register(app, NewDeps(servicio, borradorFalso{}, func() time.Time { return time.Unix(0, 0) }))
	return app
}

func TestLasTreceRutasCanonicasEstanMontadas(t *testing.T) {
	app := nuevaApp(t)
	esperadas := map[string]bool{
		"GET /api/v1/tasas/monitor": true, "GET /api/v1/tasas/socios": true,
		"GET /api/v1/tasas/hoo-borrador": true, "PUT /api/v1/tasas/socios/:socio": true,
		"POST /api/v1/tasas/guardar-lote": true, "POST /api/v1/tasas/publicar-lote": true,
		"POST /api/v1/tasas/notificar-remesas": true, "GET /api/v1/tasas/calcular/:socio": true,
		"GET /api/v1/tasas/render/:socio": true, "POST /api/v1/tasas/render": true,
		"POST /api/v1/tasas/disparar": true, "POST /api/v1/tasas/disparar-lote": true,
		"DELETE /api/v1/tasas/lote/:id_tasa": true,
	}
	encontradas := map[string]bool{}
	for _, ruta := range app.GetRoutes() {
		if strings.HasPrefix(ruta.Path, "/api/v1/tasas") && ruta.Method != "HEAD" {
			encontradas[ruta.Method+" "+ruta.Path] = true
		}
	}
	if len(encontradas) != 13 {
		t.Fatalf("se esperaban 13 rutas, hay %d: %v", len(encontradas), encontradas)
	}
	for ruta := range esperadas {
		if !encontradas[ruta] {
			t.Fatalf("falta la ruta canónica %s", ruta)
		}
	}
}

func TestSmokeDeLosTreceEndpoints(t *testing.T) {
	casos := []struct {
		metodo string
		ruta   string
		cuerpo string
		png    bool
	}{
		{"GET", "/api/v1/tasas/monitor", "", false},
		{"GET", "/api/v1/tasas/socios", "", false},
		{"GET", "/api/v1/tasas/hoo-borrador", "", false},
		{"PUT", "/api/v1/tasas/socios/ANA", `{"activo":true}`, false},
		{"POST", "/api/v1/tasas/guardar-lote", `{"tasas":{"USD":1}}`, false},
		{"POST", "/api/v1/tasas/publicar-lote", `{"modo_prueba_global":true}`, false},
		{"POST", "/api/v1/tasas/notificar-remesas", "", false},
		{"GET", "/api/v1/tasas/calcular/ANA", "", false},
		{"GET", "/api/v1/tasas/render/ANA", "", true},
		{"POST", "/api/v1/tasas/render", `{"socio":"ANA"}`, true},
		{"POST", "/api/v1/tasas/disparar", `{"socio":"ANA","modo_prueba":true}`, false},
		{"POST", "/api/v1/tasas/disparar-lote", `{"modo_prueba_global":true}`, false},
		{"DELETE", "/api/v1/tasas/lote/T060", "", false},
	}
	app := nuevaApp(t)
	for _, caso := range casos {
		t.Run(caso.metodo+" "+caso.ruta, func(t *testing.T) {
			req := httptest.NewRequest(caso.metodo, caso.ruta, strings.NewReader(caso.cuerpo))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("la petición falló: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("se esperaba 200, llegó %d", resp.StatusCode)
			}
			if caso.png {
				if tipo := resp.Header.Get("Content-Type"); !strings.Contains(tipo, "png") {
					t.Fatalf("se esperaba un PNG, llegó %q", tipo)
				}
				return
			}
			var cuerpo map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&cuerpo); err != nil {
				t.Fatalf("respuesta JSON inválida: %v", err)
			}
			if cuerpo["success"] != true {
				t.Fatalf("la respuesta no declara success: %v", cuerpo)
			}
		})
	}
}
