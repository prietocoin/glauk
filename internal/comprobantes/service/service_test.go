package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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
	profiles  map[string]bool
}

func (s *stubRepo) ListRows(context.Context, domain.ListFilters) ([]calc.SourceRow, int, error) {
	return s.listRows, len(s.listRows), nil
}
func (s *stubRepo) Fundda(context.Context) (calc.Profile, error)       { return s.fundda, nil }
func (s *stubRepo) Liquidar(context.Context, domain.Liquidacion) error { return s.liquidar }
func (s *stubRepo) ProfilesExist(_ context.Context, names ...string) (bool, error) {
	for _, name := range names {
		if !s.profiles[domain.CanonicalSocio(name)] {
			return false, nil
		}
	}
	return true, nil
}
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
	updated  store.RawUpdate
	lotID    string
	liq      domain.Liquidacion
	deleted  bool
	actual   domain.Liquidacion
	profiles map[string]calc.Profile
	upserts  int
}

func (t *stubTx) LockComprobante(context.Context, string) (store.RawComprobante, error) {
	moneda := "COP"
	return store.RawComprobante{Monto: 100, Moneda: &moneda}, nil
}
func (t *stubTx) LiquidacionActual(context.Context, string) (domain.Liquidacion, bool, error) {
	if t.actual.HashLargo == "" {
		return domain.Liquidacion{}, false, nil
	}
	return t.actual, true, nil
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
	trimmed := strings.TrimSpace(name)
	if domain.IsPlaceholderSocio(trimmed) {
		return calc.Profile{}, domain.Validationf("el socio %q no es un socio real", name)
	}
	if profile, ok := t.profiles[trimmed]; ok {
		return profile, nil
	}
	return calc.Profile{}, domain.Validationf("el socio %q no está registrado", name)
}
func (t *stubTx) Fundda(context.Context) (calc.Profile, error) { return calc.Profile{}, nil }
func (t *stubTx) UpsertLiquidacion(_ context.Context, liq domain.Liquidacion) error {
	t.liq = liq
	t.upserts++
	return nil
}
func (t *stubTx) DeleteComprobante(context.Context, string) error {
	t.deleted = true
	return nil
}

func TestListRejectsInvalidDates(t *testing.T) {
	serv := New(&stubRepo{})
	_, _, err := serv.List(context.Background(), domain.ListFilters{FechaInicio: "2026-13-40"})
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
	items, _, err := New(repo).List(context.Background(), domain.ListFilters{})
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

func TestLiquidarRechazaEquivalentesCero(t *testing.T) {
	var input dto.LiquidarInput
	if err := json.Unmarshal([]byte(`{
		"hash_largo":"HASH","socio_1":"SOCIO_A","socio_2":"SOCIO_B",
		"monto_1":1,"tasa_1":1,"me1":0,
		"monto_2":-1,"tasa_2":1,"me2":0,"lote_tasa":"T052"}`), &input); err != nil {
		t.Fatal(err)
	}
	_, err := New(&stubRepo{}).Liquidar(context.Background(), input)
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, esperado ValidationError", err)
	}
}

func TestLiquidarExigeSociosRegistrados(t *testing.T) {
	var input dto.LiquidarInput
	if err := json.Unmarshal([]byte(`{
		"hash_largo":"HASH","socio_1":"SOCIO_A","socio_2":"SOCIO_B",
		"monto_1":1,"tasa_1":1,"me1":1,
		"monto_2":-1,"tasa_2":1,"me2":-1,"lote_tasa":"T052"}`), &input); err != nil {
		t.Fatal(err)
	}
	_, err := New(&stubRepo{}).Liquidar(context.Background(), input)
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, esperado ValidationError por perfiles", err)
	}
}

func TestUpdateMetadataWithFullPayloadDoesNotTouchLiquidation(t *testing.T) {
	socio2 := "SOCIO_B"
	actual := domain.Liquidacion{
		HashLargo: "HASH", Socio1: "SOCIO_A", TipoOp1: "D-COP",
		Monto1: -100, Tasa1: 4000, ME1: -100,
		Socio2: &socio2, TipoOp2: "D-COP", Monto2: 100, Tasa2: 4000, ME2: 100,
		LoteTasa: "T052",
	}
	tx := &stubTx{actual: actual}
	repo := &stubRepo{inTx: func(ctx context.Context, fn func(store.Tx) error) error {
		return fn(tx)
	}}
	var input dto.UpdateInput
	if err := json.Unmarshal([]byte(`{
		"titular":"nuevo titular","banco":"banco","referencia":"REF-1",
		"socio_1":"SOCIO_A","socio_2":"SOCIO_B",
		"nombre_socio_1":"SOCIO_A","nombre_socio_2":"SOCIO_B",
		"tipo_manual":"D","moneda":"COP","monto":100,
		"lote_tasa_asignado":"T052","lote_tasa":"T052"}`), &input); err != nil {
		t.Fatal(err)
	}
	if _, err := New(repo).Update(context.Background(), "HASH", input); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if tx.upserts != 0 {
		t.Fatalf("el payload completo sin cambios financieros no debe upsertar: %d", tx.upserts)
	}
}

func TestUpdateMetadataDoesNotTouchLiquidation(t *testing.T) {
	socio2 := "SOCIO_B"
	actual := domain.Liquidacion{
		HashLargo: "HASH", Socio1: "SOCIO_A", TipoOp1: "D-COP",
		Monto1: -100, Tasa1: 4000, ME1: -100,
		Socio2: &socio2, TipoOp2: "D-COP", Monto2: 100, Tasa2: 4000, ME2: 100,
		LoteTasa: "T052",
	}
	tx := &stubTx{actual: actual}
	repo := &stubRepo{inTx: func(ctx context.Context, fn func(store.Tx) error) error {
		return fn(tx)
	}}
	var input dto.UpdateInput
	if err := json.Unmarshal([]byte(`{"titular":"nuevo titular","banco":"banco","referencia":"REF-1"}`),
		&input); err != nil {
		t.Fatal(err)
	}
	liq, err := New(repo).Update(context.Background(), "HASH", input)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if tx.upserts != 0 {
		t.Fatalf("la edición de metadatos no debe upsertar liquidación: %d", tx.upserts)
	}
	if tx.updated.Titular != "NUEVO TITULAR" || tx.updated.Banco != "BANCO" {
		t.Fatalf("metadatos no actualizados: %+v", tx.updated)
	}
	if liq.ME1 != actual.ME1 || liq.ME2 != actual.ME2 ||
		liq.Socio1 != actual.Socio1 || liq.LoteTasa != actual.LoteTasa {
		t.Fatalf("snapshot alterado: %+v", liq)
	}
}

func TestUpdateResolvesAliasesInsideTransaction(t *testing.T) {
	tx := &stubTx{profiles: map[string]calc.Profile{
		"SOCIO_A": {Nombre: "SOCIO_A", MonedaBase: "USDT",
			Monedas: map[string]calc.CurrencyConfig{"COP": {Tipo: "D", PolarD: "+"}}},
		"SOCIO_B": {Nombre: "SOCIO_B", MonedaBase: "USDT",
			Monedas: map[string]calc.CurrencyConfig{"COP": {Tipo: "D", PolarD: "+"}}},
	}}
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

func TestUpdateRecalculaAlCambiarSocio(t *testing.T) {
	socio2 := "SOCIO_A"
	tx := &stubTx{
		actual: domain.Liquidacion{
			HashLargo: "HASH", Socio1: "SOCIO_A", TipoOp1: "D-COP",
			Socio2: &socio2, TipoOp2: "D-COP", LoteTasa: "T052",
		},
		profiles: map[string]calc.Profile{
			"SOCIO_A": {Nombre: "SOCIO_A", MonedaBase: "USDT",
				Monedas: map[string]calc.CurrencyConfig{"COP": {
					Tipo: "D", PolarD: "+", Porcentaje: calc.Spread{Deposito: 2},
				}}},
			"SOCIO_B": {Nombre: "SOCIO_B", MonedaBase: "USDT",
				Monedas: map[string]calc.CurrencyConfig{"COP": {
					Tipo: "D", PolarD: "-", Porcentaje: calc.Spread{Deposito: 10},
				}}},
		},
	}
	repo := &stubRepo{inTx: func(ctx context.Context, fn func(store.Tx) error) error {
		return fn(tx)
	}}
	var input dto.UpdateInput
	if err := json.Unmarshal([]byte(`{"socio_1":"SOCIO_B","moneda_recibo":"COP","monto":440000}`),
		&input); err != nil {
		t.Fatal(err)
	}
	liq, err := New(repo).Update(context.Background(), "HASH", input)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if liq.Socio1 != "SOCIO_B" {
		t.Fatalf("el socio editado debe persistirse: %+v", liq)
	}
	if liq.TipoOp1 != "D-COP" || liq.LoteTasa != "T052" {
		t.Fatalf("el payload parcial no debe degradar tipo ni lote: %+v", liq)
	}
	// SOCIO_B aporta polar_D=- y 10% de comisión: 4000 * 1.10 = 4400.
	if liq.Tasa1 != 4400 || liq.Monto1 != -100 || liq.ME1 != -100 {
		t.Fatalf("recálculo con el contrato de SOCIO_B: %+v", liq)
	}
	if tx.liq.HashLargo != "HASH" || tx.liq.Socio1 != "SOCIO_B" {
		t.Fatalf("upsert no recibió el recálculo: %+v", tx.liq)
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
