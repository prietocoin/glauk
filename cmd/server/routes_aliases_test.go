package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"

	comprobantesdomain "glauk/internal/comprobantes/domain"
)

type fakeTasasQuerier struct{}

func (fakeTasasQuerier) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return pgxmock.NewRows([]string{"id_tasa", "tasas"}).Kind(), nil
}

func (fakeTasasQuerier) QueryRow(context.Context, string, ...any) pgx.Row { return nil }

func (fakeTasasQuerier) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(""), nil
}

func (stubComprobantes) Latest(context.Context) (comprobantesdomain.LoteTasa, error) {
	return comprobantesdomain.LoteTasa{
		IDTasa: "T001",
		Tasas:  map[string]float64{"USD": 1, "ARS": 1250.5},
	}, nil
}

func requireAliasHeaders(t *testing.T, headers http.Header, successor string) {
	t.Helper()
	if headers.Get("Deprecation") != "true" {
		t.Fatalf("Deprecation = %q", headers.Get("Deprecation"))
	}
	if headers.Get("Sunset") != sunsetAlias {
		t.Fatalf("Sunset = %q", headers.Get("Sunset"))
	}
	if link := headers.Get("Link"); !strings.Contains(link, successor) {
		t.Fatalf("Link = %q, se esperaba %q", link, successor)
	}
}

func TestCriticalAliasesContract(t *testing.T) {
	app := testApp(t)

	resp, body := request(t, app, http.MethodGet, "/api/reportes")
	if resp.StatusCode != 200 || !strings.HasPrefix(strings.TrimSpace(string(body)), "[") {
		t.Fatalf("/api/reportes = %d %s", resp.StatusCode, body)
	}
	requireAliasHeaders(t, resp.Header, "/api/v1/comprobantes")

	resp, body = request(t, app, http.MethodGet, "/api/reportes/filtros")
	var filtros map[string]any
	if err := json.Unmarshal(body, &filtros); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || filtros["success"] != true || filtros["roles"] == nil {
		t.Fatalf("/api/reportes/filtros = %d %v", resp.StatusCode, filtros)
	}
	requireAliasHeaders(t, resp.Header, "/api/v1/reportes/filtros")

	resp, body = request(t, app, http.MethodGet, "/api/tasas/ultimas")
	var ultimas map[string]any
	if err := json.Unmarshal(body, &ultimas); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || ultimas["id_tasa"] != "T001" || ultimas["tasas"] == nil {
		t.Fatalf("/api/tasas/ultimas = %d %v", resp.StatusCode, ultimas)
	}
	requireAliasHeaders(t, resp.Header, "/api/v1/comprobantes/tasas/ultimas")

	resp, body = request(t, app, http.MethodGet, "/api/tasas/monitor")
	var monitor map[string]any
	if err := json.Unmarshal(body, &monitor); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || monitor["success"] != true || monitor["id_tasa"] != "T000" {
		t.Fatalf("/api/tasas/monitor = %d %v", resp.StatusCode, monitor)
	}
	requireAliasHeaders(t, resp.Header, "/api/v1/tasas/monitor")

	resp, body = request(t, app, http.MethodGet, "/api/v1/atenea/reportes")
	if resp.StatusCode != 200 || !strings.HasPrefix(strings.TrimSpace(string(body)), "[") {
		t.Fatalf("/api/v1/atenea/reportes = %d %s", resp.StatusCode, body)
	}
	requireAliasHeaders(t, resp.Header, "/api/v1/comprobantes")
}

func TestLegacyMethodAndRemovalContracts(t *testing.T) {
	app := testApp(t)
	for _, path := range []string{"/disparar", "/api/tasas/disparar"} {
		resp, body := request(t, app, http.MethodGet, path)
		if resp.StatusCode != 405 || !strings.Contains(resp.Header.Get("Allow"), "POST") {
			t.Fatalf("%s = %d %q %s", path, resp.StatusCode, resp.Header.Get("Allow"), body)
		}
	}

	resp, body := request(t, app, http.MethodPost, "/api/tasas/n8n-webhook")
	if resp.StatusCode != 410 || !strings.Contains(string(body), `"code":"ROUTE_REMOVED"`) {
		t.Fatalf("n8n = %d %s", resp.StatusCode, body)
	}
	if !strings.Contains(resp.Header.Get("Link"), "/api/v1/tasas/guardar-lote") {
		t.Fatalf("Link = %q", resp.Header.Get("Link"))
	}
}

func TestAdminAndAteneaMediaAliasHeaders(t *testing.T) {
	app := testApp(t)
	resp, _ := request(t, app, http.MethodGet, "/api/admin/cola")
	requireAliasHeaders(t, resp.Header, "/api/v1/admin/cola")

	resp, _ = request(t, app, http.MethodPost, "/api/v1/atenea/whatsapp/enviar-media")
	requireAliasHeaders(t, resp.Header, "/api/v1/reportes/imagen")
}
