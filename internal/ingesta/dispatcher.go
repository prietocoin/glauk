package ingesta

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const maxBackoff = 30 * time.Second

// Dispatcher es el pool de workers con canal buffered que sustituye a BullMQ.
// No conoce HTTP ni PostgreSQL: recibe una función de proceso y otra de fallo.
type Dispatcher struct {
	queue     chan Job
	workers   int
	timeout   time.Duration
	retries   int
	retryBase time.Duration
	process   func(context.Context, Job) error
	fail      func(Job)
	log       *slog.Logger

	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	started bool
	wg      sync.WaitGroup
}

// NewDispatcher valida los mínimos para que un cero accidental no bloquee el
// arranque con un canal sin capacidad.
func NewDispatcher(queueSize, workers int, timeout time.Duration, retries int,
	retryBase time.Duration, process func(context.Context, Job) error,
	fail func(Job), log *slog.Logger) *Dispatcher {
	if queueSize < 1 {
		queueSize = 1
	}
	if workers < 1 {
		workers = 1
	}
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	if retries < 0 {
		retries = 0
	}
	if retryBase <= 0 {
		retryBase = 500 * time.Millisecond
	}
	if log == nil {
		log = slog.Default()
	}
	return &Dispatcher{
		queue: make(chan Job, queueSize), workers: workers, timeout: timeout,
		retries: retries, retryBase: retryBase, process: process, fail: fail, log: log,
	}
}

// Start lanza los workers una sola vez y hereda la cancelación del servidor.
func (d *Dispatcher) Start(parent context.Context) {
	if d == nil {
		return
	}
	d.mu.Lock()
	if d.started {
		d.mu.Unlock()
		return
	}
	d.started = true
	d.ctx, d.cancel = context.WithCancel(parent)
	d.mu.Unlock()

	for index := 0; index < d.workers; index++ {
		d.wg.Add(1)
		go d.worker(index)
	}
}

// Enqueue nunca bloquea al webhook: si el buffer está lleno devuelve false y
// el registro queda en EN_COLA para que un reenvío lo recupere.
func (d *Dispatcher) Enqueue(job Job) bool {
	if d == nil {
		return false
	}
	d.mu.Lock()
	ctx := d.ctx
	started := d.started
	d.mu.Unlock()
	if !started || ctx == nil {
		return false
	}
	select {
	case <-ctx.Done():
		return false
	default:
	}
	select {
	case d.queue <- job:
		return true
	default:
		d.log.Warn("ingesta: cola llena, trabajo descartado", "hash", job.HashLargo)
		return false
	}
}

// Close cancela los workers y espera a que terminen el trabajo en curso. No
// cierra el canal para que un Enqueue concurrente no provoque un panic.
func (d *Dispatcher) Close() {
	if d == nil {
		return
	}
	d.mu.Lock()
	cancel := d.cancel
	d.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	d.wg.Wait()
}

func (d *Dispatcher) worker(index int) {
	defer d.wg.Done()
	for {
		select {
		case <-d.ctx.Done():
			return
		case job := <-d.queue:
			d.processWithRetry(job)
		}
	}
}

func (d *Dispatcher) processWithRetry(job Job) {
	for attempt := 0; ; attempt++ {
		if d.ctx.Err() != nil {
			d.markFailed(job)
			return
		}
		ctx, cancel := context.WithTimeout(d.ctx, d.timeout)
		err := d.process(ctx, job)
		cancel()
		if err == nil {
			return
		}
		job.Intentos = attempt + 1
		if attempt >= d.retries {
			d.log.Error("ingesta: trabajo agotó reintentos",
				"hash", job.HashLargo, "intentos", job.Intentos, "error", err)
			d.markFailed(job)
			return
		}
		delay := d.retryBase << attempt
		if delay > maxBackoff {
			delay = maxBackoff
		}
		d.log.Warn("ingesta: reintento programado", "hash", job.HashLargo,
			"intento", job.Intentos, "espera", delay.String(), "error", err)
		select {
		case <-d.ctx.Done():
			d.markFailed(job)
			return
		case <-time.After(delay):
		}
	}
}

func (d *Dispatcher) markFailed(job Job) {
	if d.fail != nil {
		d.fail(job)
	}
}
