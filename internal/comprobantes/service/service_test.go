package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"glauk/internal/comprobantes/calc"
	"glauk/internal/comprobantes/domain"
	"glauk/internal/comprobantes/dto"
	"glauk/internal/comprobantes/store"
)

type stubRepo struct {
	Repository
	listRows  []calc.SourceRow
	fundda    calc.Profile
	inTx      func(context.Context, func(store.Tx) error) error
	liquidar  error
	mark      bool
	deleteErr error
}

func (s *stubRepo) ListRows(context.Context, domain.ListFilters) ([]calc.SourceRow, error) {
	return s.listRows, nil
}
func (s *stubRepo) Fundda(context.Context) (calc.Profile, error)       { return s.fundda, nil }
func (s *stubRepo) Liquidar(context.Context, domain.Liquidacion) error { return s.liquidar }
func (s *stubRepo) InTx(ctx context.Context, fn func(store.Tx) error) error {
	if s.inTx == nil {
		return nil
	}
	return s.inTx(ctx, fn)
}
func (s *stubRepo) MarkReprocessing(context.Context, string) (bool, error) {
	return s.mark, nil
}
func (s *stubRepo) DeleteQueue(context.Context, string) error { return s.deleteErr }

type stubTx struct {
	store.Tx
	updated store.RawUpdate
	lotID   string
	liq     domain.Liquidacion
	deleted bool
}

func (t *stubTx) LockComprobante(context.Context, string) (store.RawComprobante, error) {
	moneda := "COP"
	return store.RawComprobante{Monto: 100, Moneda: &moneda}, nil
}
func (t *stubTx) UpdateComprobante(_ context.Context, _ string, data store.RawUpdate) error {
	t.updated = data
	return nil
}
func (t *stubTx) RateLotByID(_ context.Context, id string) (calc.RateLot, error) {
	t.lotID = id
	return calc.RateLot{IDTasa: id, Tasas: map[string]float64{"COP": 4000, "USDT": 1}}, nil
}
func (t *stubTx) ProfileByName(_ context.Context, name string) (calc.Profile, error) {
	return calc.Profile{Nombre: name, MonedaBase: "USDT",
		Monedas: map[string]calc.CurrencyConfig{"COP": {Polaridad: "+"}}}, nil
}
func (t *stubTx) Fundda(context.Context) (calc.Profile, error) { return calc.Profile{}, nil }
func (t *stubTx) UpsertLiquidacion(_ context.Context, liq domain.Liquidacion) error {
	t.liq = liq
	return nil
}
func (t *stubTx) DeleteComprobante(context.Context, string) error {
	t.deleted = true
	return nil
}

func TestListRejectsInvalidDates(t *testing.T) {
	serv := New(&stubRepo{})
	_, err := serv.List(context.Background(), domain.ListFilters{FechaInicio: "2026-13-40"})
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, esperado ValidationError", err)
	}
}

func TestListProjectsRows(t *testing.T) {
	repo := &stubRepo{
		listRows: []calc.SourceRow{{HashLargo: "h1", Monto: 100, Moneda: "COP"}},
		fundda:   calc.Profile{},
	}
	items, err := New(repo).List(context.Background(), domain.ListFilters{})
	if err != nil || len(items) != 1 || items[0].HashLargo != "h1" {
		t.Fatalf("items=%+v err=%v", items, err)
	}
}

func TestLiquidarRequiresHash(t *testing.T) {
	_, err := New(&stubRepo{}).Liquidar(context.Background(), dto.LiquidarInput{})
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v", err)
	}
}

func TestUpdateResolvesAliasesInsideTransaction(t *testing.T) {
	tx := &stubTx{}
	repo := &stubRepo{inTx: func(ctx context.Context, fn func(store.Tx) error) error {
		return fn(tx)
	}}
	var input dto.UpdateInput
	if err := json.Unmarshal([]byte(`{"socio1":"SOCIO_A","socio2":"SOCIO_B",
		"moneda_recibo":"COP","id_tasa":"T052","tipo_op":"D","monto":400000}`), &input); err != nil {
		t.Fatal(err)
	}
	liq, err := New(repo).Update(context.Background(), "HASH", input)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if tx.lotID != "T052" || tx.updated.Moneda != "COP" {
		t.Fatalf("alias no usados: lote=%q moneda=%q", tx.lotID, tx.updated.Moneda)
	}
	if liq.Socio1 != "SOCIO_A" || liq.TipoOp1 != "D-COP" || liq.Monto1 != 100 {
		t.Fatalf("snapshot inesperado: %+v", liq)
	}
	if tx.liq.HashLargo != "HASH" {
		t.Fatalf("upsert no ejecutado: %+v", tx.liq)
	}
}

func TestDeleteAndReleerErrors(t *testing.T) {
	tx := &stubTx{}
	repo := &stubRepo{inTx: func(ctx context.Context, fn func(store.Tx) error) error {
		return fn(tx)
	}}
	if err := New(repo).Delete(context.Background(), "HASH"); err != nil || !tx.deleted {
		t.Fatalf("delete err=%v called=%v", err, tx.deleted)
	}
	err := New(&stubRepo{mark: false}).Releer(context.Background(), "HASH")
	if !errors.Is(err, domain.ErrComprobanteNotFound) {
		t.Fatalf("releer error = %v", err)
	}
}
