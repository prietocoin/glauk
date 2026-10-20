package postgres

import (
	"context"
	"testing"
	"time"
)

type pingerFunc func(context.Context) error

func (f pingerFunc) Ping(ctx context.Context) error {
	return f(ctx)
}

func TestPingSetsDeadline(t *testing.T) {
	var deadline time.Time
	target := pingerFunc(func(ctx context.Context) error {
		deadline, _ = ctx.Deadline()
		return nil
	})
	if err := ping(context.Background(), target, 200*time.Millisecond); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if deadline.IsZero() {
		t.Fatal("el contexto no tenía deadline")
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > 250*time.Millisecond {
		t.Fatalf("deadline fuera de rango: %v", remaining)
	}
}

func TestPingRejectsNilPoolAndInvalidTimeout(t *testing.T) {
	if err := Ping(context.Background(), nil, time.Second); err == nil {
		t.Fatal("se esperaba error por pool nil")
	}
	target := pingerFunc(func(context.Context) error { return nil })
	if err := ping(context.Background(), target, 0); err == nil {
		t.Fatal("se esperaba error por timeout no positivo")
	}
}
