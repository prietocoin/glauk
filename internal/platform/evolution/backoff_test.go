package evolution

import (
	"context"
	"errors"
	"testing"
	"time"

	"glauk/internal/platform/config"
)

func TestBackoffExponentialAndJitter(t *testing.T) {
	client := New(config.EvolutionConfig{RetryBase: 100 * time.Millisecond})
	client.randFloat = func() float64 { return 0.5 }

	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{5, 1600 * time.Millisecond},
	}
	for _, tc := range cases {
		if got := client.backoff(tc.attempt, nil); got != tc.want {
			t.Errorf("attempt %d = %v, se esperaba %v", tc.attempt, got, tc.want)
		}
	}

	client.randFloat = func() float64 { return 0 }
	if got := client.backoff(1, nil); got != 80*time.Millisecond {
		t.Errorf("jitter mínimo = %v", got)
	}
	client.randFloat = func() float64 { return 1 }
	if got := client.backoff(1, nil); got != 120*time.Millisecond {
		t.Errorf("jitter máximo = %v", got)
	}
}

func TestBackoffRetryAfterAndCap(t *testing.T) {
	client := New(config.EvolutionConfig{RetryBase: time.Second})
	if got := client.backoff(1, &APIError{RetryAfter: 2}); got != 2*time.Second {
		t.Fatalf("Retry-After = %v", got)
	}
	if got := client.backoff(1, &APIError{RetryAfter: 10}); got != maxBackoff {
		t.Fatalf("Retry-After sin tope = %v", got)
	}
	if got := client.backoff(4, nil); got != maxBackoff {
		t.Fatalf("backoff sin tope = %v", got)
	}
}

func TestBackoffDefaultBase(t *testing.T) {
	client := New(config.EvolutionConfig{})
	client.randFloat = func() float64 { return 0.5 }
	if got := client.backoff(1, nil); got != 250*time.Millisecond {
		t.Fatalf("base por defecto = %v", got)
	}
}

func TestWaitContextHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitContext(ctx, time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := waitContext(ctx, time.Second); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v", err)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("espera no cancelada: %v", elapsed)
	}
}
