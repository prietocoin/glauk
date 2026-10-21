package reportes

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestIdempotencyKeyDerivaYValida(t *testing.T) {
	store := NewIdempotencyStore(time.Minute)
	explicit := store.Key(" clave-123 ", "1@s.whatsapp.net", "hola", "abc")
	if explicit != "clave-123" {
		t.Fatalf("clave explícita = %q", explicit)
	}
	derivedA := store.Key("", "1@s.whatsapp.net", "hola", "abc")
	derivedB := store.Key("", "1@s.whatsapp.net", "hola", "abc")
	if derivedA == "" || derivedA != derivedB {
		t.Fatalf("clave derivada inestable: %q / %q", derivedA, derivedB)
	}
	if got := store.Key("corta", "1@s.whatsapp.net", "hola", "abc"); got != derivedA {
		t.Fatalf("clave inválida debe derivarse: %q", got)
	}
}

func TestIdempotencyCicloYDuplicado(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	store := NewIdempotencyStore(time.Minute)
	store.now = func() time.Time { return now }

	if _, _, err := store.Begin(""); !errors.Is(err, ErrEnvioEnCurso) {
		t.Fatalf("clave vacía: %v", err)
	}
	if _, created, err := store.Begin("abc12345"); err != nil || !created {
		t.Fatalf("primer begin: created=%v err=%v", created, err)
	}
	if _, _, err := store.Begin("abc12345"); !errors.Is(err, ErrEnvioEnCurso) {
		t.Fatalf("segundo begin: %v", err)
	}
	store.Abort("abc12345")
	if _, created, err := store.Begin("abc12345"); err != nil || !created {
		t.Fatalf("begin tras abort: created=%v err=%v", created, err)
	}
	store.Commit("abc12345", Resultado{MessageID: "m1", Estado: "enviado"})
	got, created, err := store.Begin("abc12345")
	if err != nil || created || got == nil || got.MessageID != "m1" {
		t.Fatalf("duplicado: got=%+v created=%v err=%v", got, created, err)
	}
	store.Commit("", Resultado{})
	store.Abort("")
}

func TestIdempotencyExpiraYPurga(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	store := NewIdempotencyStore(time.Minute)
	store.now = func() time.Time { return now }
	store.max = 1

	if _, created, _ := store.Begin("primera1"); !created {
		t.Fatal("no reservó la primera clave")
	}
	if _, _, err := store.Begin("segunda2"); !errors.Is(err, ErrIdempotenciaLlena) {
		t.Fatalf("límite: %v", err)
	}
	store.Commit("primera1", Resultado{MessageID: "m1"})
	now = now.Add(2 * time.Minute)
	if _, created, err := store.Begin("segunda2"); err != nil || !created {
		t.Fatalf("purga de expirada: created=%v err=%v", created, err)
	}
	store.Abort("segunda2")
	now = now.Add(3 * time.Minute)
	if _, created, err := store.Begin("tercera3"); err != nil || !created {
		t.Fatalf("reserva tras abort: created=%v err=%v", created, err)
	}
	if store.ttl != time.Minute {
		t.Fatalf("ttl = %v", store.ttl)
	}
	if NewIdempotencyStore(0).ttl != defaultIdempotencyTTL {
		t.Fatal("TTL por defecto no aplicado")
	}
}

func TestRateLimiterRafagaYRecarga(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	limiter := NewRateLimiter(60)
	limiter.now = func() time.Time { return now }

	for i := 0; i < 10; i++ {
		if !limiter.Allow(" 10.0.0.1 ") {
			t.Fatalf("token %d rechazado", i+1)
		}
	}
	if limiter.Allow("10.0.0.1") {
		t.Fatal("la ráfaga 11 debe rechazarse")
	}
	if got := limiter.RetryAfter("10.0.0.1"); got != 1 {
		t.Fatalf("retry = %d", got)
	}
	if got := limiter.RetryAfter("10.0.0.2"); got != 0 {
		t.Fatalf("retry IP nueva = %d", got)
	}
	now = now.Add(2 * time.Second)
	if !limiter.Allow("10.0.0.1") {
		t.Fatal("dos segundos deben reponer dos tokens")
	}
	if got := limiter.RetryAfter("10.0.0.1"); got != 0 {
		t.Fatalf("retry con token restante = %d", got)
	}
}

func TestRateLimiterIPVaciaYDeshabilitado(t *testing.T) {
	limiter := NewRateLimiter(60)
	for i := 0; i < 10; i++ {
		if !limiter.Allow("") {
			t.Fatalf("token vacío %d", i+1)
		}
	}
	if limiter.Allow("unknown") {
		t.Fatal("IP vacía y unknown deben compartir bucket")
	}
	var nilLimiter *RateLimiter
	if !nilLimiter.Allow("x") || nilLimiter.RetryAfter("x") != 0 {
		t.Fatal("limiter nil debe permitir")
	}
	disabled := &RateLimiter{}
	if !disabled.Allow("x") || disabled.RetryAfter("x") != 0 {
		t.Fatal("limiter sin cuota debe permitir")
	}
	if NewRateLimiter(0).perMinute != defaultRateLimitPerMinute {
		t.Fatal("cuota por defecto no aplicada")
	}
}

func TestModuleYFiltrosVista(t *testing.T) {
	if NewModule(nil).Valido() {
		t.Fatal("módulo nil no puede ser válido")
	}
	var nilModule *Module
	if nilModule.Valido() {
		t.Fatal("módulo nil no puede ser válido")
	}
	movs := &fakeMovimientos{filtros: Filtros{Roles: []string{"ADMIN"}}}
	service := testService(&fakeSocios{}, movs, &fakeSender{})
	module := NewModule(service)
	if !module.Valido() || module.Idempotency == nil || module.RateLimiter == nil {
		t.Fatalf("módulo incompleto: %+v", module)
	}
	got, err := module.Filtros(context.Background(), " ADMIN ")
	if err != nil || len(got.Roles) != 1 || movs.rol != "ADMIN" {
		t.Fatalf("filtros: %+v err=%v rol=%q", got, err, movs.rol)
	}

	original := Filtros{Roles: []string{"A"}, Entidades: []string{"B"}, Hashes: []string{"C"}}
	copy := FiltrosVista(original)
	copy.Roles[0] = "MUTADO"
	copy.Entidades[0] = "MUTADO"
	copy.Hashes[0] = "MUTADO"
	if original.Roles[0] != "A" || original.Entidades[0] != "B" || original.Hashes[0] != "C" {
		t.Fatalf("copia defensiva falló: %+v", original)
	}
}
