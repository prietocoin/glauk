package ingesta

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestDispatcherReintentaHastaExito(t *testing.T) {
	var calls atomic.Int32
	done := make(chan struct{})
	process := func(context.Context, Job) error {
		if calls.Add(1) < 2 {
			return errors.New("fallo transitorio")
		}
		close(done)
		return nil
	}
	dispatcher := NewDispatcher(4, 1, time.Second, 3, time.Millisecond,
		process, nil, testLogger())
	dispatcher.Start(context.Background())
	defer dispatcher.Close()

	if !dispatcher.Enqueue(Job{Event: Event{HashLargo: "hash"}}) {
		t.Fatal("el trabajo no se encoló")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout esperando el reintento")
	}
	if calls.Load() != 2 {
		t.Fatalf("llamadas = %d, want 2", calls.Load())
	}
}

func TestDispatcherAgotaReintentos(t *testing.T) {
	failed := make(chan struct{})
	process := func(context.Context, Job) error { return errors.New("fallo persistente") }
	fail := func(Job) { close(failed) }
	dispatcher := NewDispatcher(1, 1, time.Second, 1, time.Millisecond,
		process, fail, testLogger())
	dispatcher.Start(context.Background())
	defer dispatcher.Close()

	if !dispatcher.Enqueue(Job{Event: Event{HashLargo: "hash"}}) {
		t.Fatal("el trabajo no se encoló")
	}
	select {
	case <-failed:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout esperando el fallo final")
	}
}

func TestDispatcherEnqueueAntesDeStart(t *testing.T) {
	dispatcher := NewDispatcher(1, 1, time.Second, 0, time.Millisecond,
		func(context.Context, Job) error { return nil }, nil, testLogger())
	if dispatcher.Enqueue(Job{}) {
		t.Fatal("no debe aceptar trabajos antes de Start")
	}
}

func TestDispatcherCloseDetieneWorkers(t *testing.T) {
	dispatcher := NewDispatcher(1, 2, time.Second, 0, time.Millisecond,
		func(context.Context, Job) error { return nil }, nil, testLogger())
	dispatcher.Start(context.Background())
	dispatcher.Close()
	if dispatcher.Enqueue(Job{}) {
		t.Fatal("no debe aceptar trabajos después de Close")
	}
}
