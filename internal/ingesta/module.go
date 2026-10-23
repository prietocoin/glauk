package ingesta

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"glauk/internal/platform/config"
	"glauk/internal/platform/evolution"
	"glauk/internal/platform/gemini"
	"glauk/internal/platform/r2"
)

// ErrQueueFull se responde como 503 para que Evolution reintente el webhook.
var ErrQueueFull = errors.New("ingesta: cola de trabajos saturada")

// Module agrupa el caso de uso y su pool de workers. cmd/server solo llama a
// Start, Close y Recibir; no conoce canales ni clientes externos.
type Module struct {
	service *Service
	queue   *Dispatcher
	enabled bool
}

// NewModule cablea R2, Gemini, el store y el liquidador. Se construye siempre
// para que la ruta exista; Enabled decide si atiende o responde 503.
func NewModule(cfg *config.Config, pool *pgxpool.Pool, evo *evolution.Client,
	log *slog.Logger) *Module {
	if cfg == nil || pool == nil {
		return nil
	}
	repository := NewRepository(pool, log)
	images := r2.New(cfg.Ingesta.R2)
	extractor := gemini.New(cfg.Ingesta.Gemini)
	liquidador := NewLiquidador(pool)

	var media MediaFetcher
	if evo != nil {
		media = evo
	}
	service := NewService(repository, media, images, extractor, liquidador,
		cfg.Ingesta, cfg.Evolution.Instance, log)
	queue := NewDispatcher(cfg.Ingesta.QueueSize, cfg.Ingesta.Workers,
		cfg.Ingesta.JobTimeout, cfg.Ingesta.MaxRetries, cfg.Ingesta.RetryBase,
		service.Procesar, service.Fallar, log)

	return &Module{service: service, queue: queue, enabled: cfg.Ingesta.Enabled}
}

// Enabled refleja INGESTA_ENABLED. Deshabilitado no arranca workers.
func (m *Module) Enabled() bool { return m != nil && m.enabled }

// Start lanza el pool de workers con el contexto del servidor.
func (m *Module) Start(ctx context.Context) {
	if m == nil || !m.enabled {
		return
	}
	m.queue.Start(ctx)
}

// Close espera a que los workers terminen el trabajo en curso.
func (m *Module) Close() {
	if m == nil {
		return
	}
	m.queue.Close()
}

// Recibir valida el binomio y encola sin bloquear la respuesta HTTP.
func (m *Module) Recibir(ctx context.Context, raw []byte) (Decision, error) {
	if m == nil || !m.enabled {
		return Decision{}, errors.New("ingesta deshabilitada")
	}
	decision, job, err := m.service.Recibir(ctx, raw)
	if err != nil {
		return Decision{}, err
	}
	if job != nil && !m.queue.Enqueue(*job) {
		m.service.Fallar(*job)
		return Decision{}, ErrQueueFull
	}
	return decision, nil
}
