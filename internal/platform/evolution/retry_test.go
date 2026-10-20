package evolution

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"glauk/internal/platform/config"
)

func TestRetryAfterIsRespected(t *testing.T) {
	var calls atomic.Int32
	var delays []time.Duration
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"key":{"id":"AFTER-RETRY"}}`))
	}, config.EvolutionConfig{MaxRetries: 1, RetryBase: time.Millisecond})
	client.sleep = func(_ context.Context, delay time.Duration) error {
		delays = append(delays, delay)
		return nil
	}

	receipt, err := client.SendText(context.Background(), TextInput{Number: "123", Text: "hola"})
	if err != nil {
		t.Fatalf("SendText: %v", err)
	}
	if receipt.MessageID != "AFTER-RETRY" || calls.Load() != 2 {
		t.Fatalf("recibo=%+v llamadas=%d", receipt, calls.Load())
	}
	if len(delays) != 1 || delays[0] != time.Second {
		t.Fatalf("delays = %v", delays)
	}
}

func TestTransientStatusesRetryUntilMax(t *testing.T) {
	for _, status := range []int{408, 425, 500, 502, 503, 504} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var calls atomic.Int32
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.WriteHeader(status)
			}, config.EvolutionConfig{MaxRetries: 2, RetryBase: time.Millisecond})
			client.sleep = func(context.Context, time.Duration) error { return nil }

			_, err := client.SendText(context.Background(), TextInput{Number: "123", Text: "hola"})
			var apiErr *APIError
			if !errors.As(err, &apiErr) || !apiErr.Retryable {
				t.Fatalf("error = %v", err)
			}
			if calls.Load() != 3 {
				t.Fatalf("llamadas = %d, se esperaban 3", calls.Load())
			}
		})
	}
}

func TestNetworkTimeoutIsRetryable(t *testing.T) {
	var calls atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}, config.EvolutionConfig{Timeout: 20 * time.Millisecond, MaxRetries: 1, RetryBase: time.Millisecond})
	client.sleep = func(context.Context, time.Duration) error { return nil }

	_, err := client.SendText(context.Background(), TextInput{Number: "123", Text: "hola"})
	if err == nil {
		t.Fatal("se esperaba timeout de red")
	}
	if calls.Load() != 2 {
		t.Fatalf("llamadas = %d, el timeout debía reintentarse", calls.Load())
	}
}

func TestContextCancelledDuringBackoff(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}, config.EvolutionConfig{MaxRetries: 3, RetryBase: time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client.sleep = func(_ context.Context, _ time.Duration) error {
		cancel()
		return context.Canceled
	}

	_, err := client.SendText(ctx, TextInput{Number: "123", Text: "hola"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}
