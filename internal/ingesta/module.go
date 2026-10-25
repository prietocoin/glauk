package ingesta

import (
	"context"
	"errors"
	"log/slog"
	"time"

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
	service    *Service
	store      *Repository
	queue      *Dispatcher
	enabled    bool
	log        *slog.Logger
	staleAfter time.Duration
}

// NewModule cablea R2, Gemini, el store y el liquidador. Se construye siempre
// para que la ruta exista; Enabled decide si atiende o responde 503.
func NewModule(cfg *config.Config, pool *pgxpool.Pool, evo *evolution.Client,
	log *slog.Logger) *Module {
	if cfg == nil || pool == nil {
		return nil
	}
	if log == nil {
		log = slog.Default()
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

	staleAfter := cfg.Ingesta.StaleAfter
	if staleAfter <= 0 {
		staleAfter = 15 * time.Minute
	}
	return &Module{
		service: service, store: repository, queue: queue,
		enabled: cfg.Ingesta.Enabled, log: log, staleAfter: staleAfter,
	}
}

// Enabled refleja INGESTA_ENABLED. Deshabilitado no arranca workers.
func (m *Module) Enabled() bool { return m != nil && m.enabled }

// Start sanea el estado persistido antes de lanzar el pool de workers. El
// barrido corre antes de abrir el puerto HTTP, de modo que ningún webhook
// compite con la reparación de hashes ni con la transición de EN_COLA.
func (m *Module) Start(ctx context.Context) {
	if m == nil || !m.enabled {
		return
	}
	if m.store != nil {
		if normalizados, err := m.store.NormalizeRegistrosHash(ctx); err != nil {
			m.log.Warn("ingesta: no se pudieron normalizar hashes históricos",
				"error", err)
		} else if normalizados > 0 {
			m.log.Info("ingesta: hashes históricos normalizados",
				"cantidad", normalizados)
		}
		if barridos, err := m.store.SweepStaleJobs(ctx, m.staleAfter); err != nil {
			m.log.Warn("ingesta: no se pudieron barrer trabajos EN_COLA",
				"error", err)
		} else if barridos > 0 {
			m.log.Warn("ingesta: trabajos EN_COLA marcados como FALLO",
				"cantidad", barridos)
		}
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
