package http

import (
	"context"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/comprobantes/domain"
	"glauk/internal/comprobantes/dto"
	"glauk/internal/platform/config"
)

type stubService struct {
	list    func(context.Context, domain.ListFilters) ([]domain.Comprobante, error)
	latest  func(context.Context) (domain.LoteTasa, error)
	history func(context.Context) ([]domain.LoteTasa, error)
	liquid  func(context.Context, dto.LiquidarInput) (domain.Liquidacion, error)
	update  func(context.Context, string, dto.UpdateInput) (domain.Liquidacion, error)
	del     func(context.Context, string) error
	releer  func(context.Context, string) error
	queue   func(context.Context) ([]domain.QueueItem, error)
	uq      func(context.Context, string, dto.UpdateInput) (domain.QueueUpdate, error)
	dq      func(context.Context, string) error
}

func (s stubService) List(ctx context.Context, f domain.ListFilters) ([]domain.Comprobante, error) {
	if s.list != nil {
		return s.list(ctx, f)
	}
	return []domain.Comprobante{}, nil
}
func (s stubService) Latest(ctx context.Context) (domain.LoteTasa, error) {
	if s.latest != nil {
		return s.latest(ctx)
	}
	return domain.LoteTasa{IDTasa: "T001"}, nil
}
func (s stubService) History(ctx context.Context) ([]domain.LoteTasa, error) {
	if s.history != nil {
		return s.history(ctx)
	}
	return []domain.LoteTasa{}, nil
}
func (s stubService) Liquidar(ctx context.Context, in dto.LiquidarInput) (domain.Liquidacion, error) {
	if s.liquid != nil {
		return s.liquid(ctx, in)
	}
	return domain.Liquidacion{HashLargo: in.HashLargo}, nil
}
func (s stubService) Update(ctx context.Context, hash string, in dto.UpdateInput) (domain.Liquidacion, error) {
	if s.update != nil {
		return s.update(ctx, hash, in)
	}
	return domain.Liquidacion{HashLargo: hash}, nil
}
func (s stubService) Delete(ctx context.Context, hash string) error {
	if s.del != nil {
		return s.del(ctx, hash)
	}
	return nil
}
func (s stubService) Releer(ctx context.Context, hash string) error {
	if s.releer != nil {
		return s.releer(ctx, hash)
	}
	return nil
}
func (s stubService) Queue(ctx context.Context) ([]domain.QueueItem, error) {
	if s.queue != nil {
		return s.queue(ctx)
	}
	return []domain.QueueItem{}, nil
}
func (s stubService) UpdateQueue(ctx context.Context, hash string, in dto.UpdateInput) (domain.QueueUpdate, error) {
	if s.uq != nil {
		return s.uq(ctx, hash, in)
	}
	return domain.QueueUpdate{}, nil
}
func (s stubService) DeleteQueue(ctx context.Context, hash string) error {
	if s.dq != nil {
		return s.dq(ctx, hash)
	}
	return nil
}

func newTestApp(t *testing.T, key string) *fiber.App {
	t.Helper()
	app := fiber.New()
	cfg := &config.Config{Admin: config.AdminConfig{APIKey: key}}
	if err := Register(app, NewDeps(stubService{}, cfg)); err != nil {
		t.Fatalf("Register: %v", err)
	}
	return app
}

func request(t *testing.T, app *fiber.App, method, path, body string) *nethttp.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test %s %s: %v", method, path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func TestAdminGuardStatusesAndNoStore(t *testing.T) {
	disabled := newTestApp(t, "")
	if resp := request(t, disabled, nethttp.MethodGet, "/api/v1/admin/cola", ""); resp.StatusCode != 503 {
		t.Fatalf("clave de entorno vacía = %d", resp.StatusCode)
	}

	app := newTestApp(t, "secret")
	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/admin/cola", nil)
	req.Header.Set("X-Admin-Key", "wrong")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("clave incorrecta = %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	req = httptest.NewRequest(nethttp.MethodGet, "/api/v1/admin/cola", nil)
	req.Header.Set("X-Admin-Key", "secret")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || resp.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("clave válida status=%d cache=%q", resp.StatusCode, resp.Header.Get("Cache-Control"))
	}
}

func TestListInvalidDateIs400(t *testing.T) {
	app := newTestApp(t, "secret")
	resp := request(t, app, nethttp.MethodGet,
		"/api/v1/comprobantes?fechaInicio=2026-99-99", "")
	if resp.StatusCode != 400 {
		t.Fatalf("fecha inválida = %d", resp.StatusCode)
	}
}

func TestCanonicalRoutesSmoke(t *testing.T) {
	app := newTestApp(t, "secret")
	cases := []struct {
		method string
		path   string
		body   string
	}{
		{nethttp.MethodGet, "/api/v1/comprobantes", ""},
		{nethttp.MethodPost, "/api/v1/comprobantes/liquidar", `{}`},
		{nethttp.MethodPost, "/api/v1/comprobantes/HASH/releer", ""},
		{nethttp.MethodPut, "/api/v1/comprobantes/HASH", `{}`},
		{nethttp.MethodDelete, "/api/v1/comprobantes/HASH", ""},
		{nethttp.MethodGet, "/api/v1/comprobantes/tasas/ultimas", ""},
		{nethttp.MethodGet, "/api/v1/comprobantes/tasas/historial", ""},
	}
	for _, tc := range cases {
		resp := request(t, app, tc.method, tc.path, tc.body)
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("%s %s = %d: %s", tc.method, tc.path, resp.StatusCode, body)
		}
	}
}

func TestAdminMutationsWithValidKey(t *testing.T) {
	app := newTestApp(t, "secret")
	for _, tc := range []struct{ method, path, body string }{
		{nethttp.MethodPut, "/api/v1/admin/cola/HASH", `{}`},
		{nethttp.MethodDelete, "/api/v1/admin/cola/HASH", ""},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Admin-Key", "secret")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("%s %s = %d", tc.method, tc.path, resp.StatusCode)
		}
		_ = resp.Body.Close()
	}
}
