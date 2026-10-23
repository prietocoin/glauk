package ingesta

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"glauk/internal/comprobantes/domain"
	"glauk/internal/platform/config"
	"glauk/internal/platform/gemini"
)

type fakeStore struct {
	registro Registro
	claim    bool
	contexto Contexto
	imagen   string
	err      error

	input            LiquidacionInput
	estado           Estado
	updatedURL       string
	savedExtraction  bool
	savedLiquidation bool
}

func (f *fakeStore) Registrar(context.Context, Event) (Registro, error)      { return f.registro, f.err }
func (f *fakeStore) Claim(context.Context, string) (bool, error)             { return f.claim, f.err }
func (f *fakeStore) ImagenExistente(context.Context, string) (string, error) { return f.imagen, f.err }
func (f *fakeStore) ActualizarImagen(_ context.Context, _, url string) error {
	f.updatedURL = url
	return f.err
}
func (f *fakeStore) Contexto(context.Context, string) (Contexto, error) { return f.contexto, f.err }
func (f *fakeStore) GuardarExtraccion(_ context.Context, _, _, _ string,
	_ gemini.Result) error {
	f.savedExtraction = true
	return f.err
}
func (f *fakeStore) GuardarLiquidacion(_ context.Context, input LiquidacionInput,
	_ LiquidacionResult, _ bool) error {
	f.input = input
	f.savedLiquidation = true
	return f.err
}
func (f *fakeStore) MarcarEstado(_ context.Context, _ string, estado Estado) error {
	f.estado = estado
	return f.err
}

type fakeMedia struct {
	data []byte
	mime string
	err  error
}

func (f fakeMedia) DownloadMedia(context.Context, string, json.RawMessage,
	json.RawMessage) ([]byte, string, error) {
	return f.data, f.mime, f.err
}

type fakeImages struct {
	putErr error
	putURL string
	puts   int
}

func (f *fakeImages) Put(_ context.Context, key string, _ []byte, _ string) (string, error) {
	f.puts++
	if f.putErr != nil {
		return "", f.putErr
	}
	if f.putURL != "" {
		return f.putURL, nil
	}
	return "https://cdn.example/" + key, nil
}
func (f *fakeImages) PublicURL(key string) string { return key }

type fakeAI struct {
	result gemini.Result
	err    error
	calls  int
}

func (f *fakeAI) Extract(context.Context, string, []byte, string) (gemini.Result, error) {
	f.calls++
	return f.result, f.err
}

type fakeLiq struct {
	result LiquidacionResult
	err    error
	calls  int
}

func (f *fakeLiq) Liquidar(context.Context, LiquidacionInput) (LiquidacionResult, error) {
	f.calls++
	return f.result, f.err
}

func testService(store *fakeStore, media MediaFetcher, images ImageStore,
	ai Extractor, liq Liquidador) *Service {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewService(store, media, images, ai, liq, config.IngestaConfig{}, "glauk", log)
}

func TestRecibirWaitingPair(t *testing.T) {
	store := &fakeStore{registro: Registro{Conteo: 1}}
	service := testService(store, fakeMedia{}, &fakeImages{}, &fakeAI{}, &fakeLiq{})
	decision, job, err := service.Recibir(context.Background(), imagePayload(indexedSha(), "glauk"))
	if err != nil {
		t.Fatalf("recibir: %v", err)
	}
	if decision.Status != "waiting_pair" || job != nil {
		t.Fatalf("decision = %+v, job = %v", decision, job)
	}
}

func TestRecibirQueuesCompletePair(t *testing.T) {
	store := &fakeStore{registro: Registro{Conteo: 2, Completo: true}, claim: true}
	service := testService(store, fakeMedia{}, &fakeImages{}, &fakeAI{}, &fakeLiq{})
	decision, job, err := service.Recibir(context.Background(), imagePayload(indexedSha(), "glauk"))
	if err != nil {
		t.Fatalf("recibir: %v", err)
	}
	if decision.Status != "queued" || job == nil {
		t.Fatalf("decision = %+v, job = %v", decision, job)
	}
	if job.HashLargo != decision.Hash {
		t.Fatalf("hash del job = %s, decision = %s", job.HashLargo, decision.Hash)
	}
}

func TestRecibirDuplicate(t *testing.T) {
	store := &fakeStore{registro: Registro{Conteo: 2}, claim: false}
	service := testService(store, fakeMedia{}, &fakeImages{}, &fakeAI{}, &fakeLiq{})
	decision, job, err := service.Recibir(context.Background(), imagePayload(indexedSha(), "glauk"))
	if err != nil || decision.Status != "duplicate" || job != nil {
		t.Fatalf("decision = %+v, job = %v, err = %v", decision, job, err)
	}
}

func TestProcesarPipelineCompleto(t *testing.T) {
	store := &fakeStore{contexto: Contexto{HashCorto: "ABC", Grupos: []string{"g1", "g2"}}}
	images := &fakeImages{}
	ai := &fakeAI{result: gemini.Result{Monto: 100, Moneda: "USD", Banco: "BANCO"}}
	liq := &fakeLiq{result: LiquidacionResult{
		Snapshot: domain.Liquidacion{HashLargo: "hash", Socio1: "ANA"},
	}}
	service := testService(store, fakeMedia{data: []byte("imagen"), mime: "image/jpeg"},
		images, ai, liq)

	job := Job{Event: Event{
		HashLargo: "hash", HashCorto: "ABC", Instancia: "glauk",
		Key: json.RawMessage(`{"id":"1"}`), Message: json.RawMessage(`{}`),
	}}
	if err := service.Procesar(context.Background(), job); err != nil {
		t.Fatalf("procesar: %v", err)
	}
	if !store.savedExtraction || !store.savedLiquidation {
		t.Fatalf("extraccion=%v liquidacion=%v", store.savedExtraction, store.savedLiquidation)
	}
	if ai.calls != 1 || images.puts != 1 {
		t.Fatalf("gemini=%d puts=%d", ai.calls, images.puts)
	}
	if store.estado != EstadoProcesado {
		t.Fatalf("estado = %s", store.estado)
	}
	if store.input.Monto != 100 || store.input.URLImagen == "" {
		t.Fatalf("input = %+v", store.input)
	}
}

func TestProcesarReutilizaExtraccion(t *testing.T) {
	store := &fakeStore{contexto: Contexto{
		Procesado: true, Monto: 50, Moneda: "USD", Grupos: []string{"g1", "g2"},
	}}
	ai := &fakeAI{}
	liq := &fakeLiq{result: LiquidacionResult{Snapshot: domain.Liquidacion{HashLargo: "hash"}}}
	service := testService(store, fakeMedia{data: []byte("imagen")}, &fakeImages{}, ai, liq)

	job := Job{Event: Event{
		HashLargo: "hash", Key: json.RawMessage(`{"id":"1"}`), Message: json.RawMessage(`{}`),
	}}
	if err := service.Procesar(context.Background(), job); err != nil {
		t.Fatalf("procesar: %v", err)
	}
	if ai.calls != 0 || store.savedExtraction {
		t.Fatalf("gemini=%d extraccion=%v", ai.calls, store.savedExtraction)
	}
	if liq.calls != 1 || store.input.Monto != 50 {
		t.Fatalf("liquidacion=%d input=%+v", liq.calls, store.input)
	}
}

func TestProcesarFallaSinImagen(t *testing.T) {
	store := &fakeStore{}
	service := testService(store, fakeMedia{err: errors.New("sin media")},
		&fakeImages{}, &fakeAI{}, &fakeLiq{})
	job := Job{Event: Event{
		HashLargo: "hash", Key: json.RawMessage(`{"id":"1"}`), Message: json.RawMessage(`{}`),
	}}
	if err := service.Procesar(context.Background(), job); err == nil {
		t.Fatal("se esperaba error sin imagen")
	}
	if store.savedLiquidation {
		t.Fatal("no debe liquidar sin imagen")
	}
}
