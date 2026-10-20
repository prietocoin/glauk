package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"glauk/internal/platform/config"
)

func validDBConfig() config.DBConfig {
	return config.DBConfig{
		DatabaseURL:      "postgres://user:pass@127.0.0.1:5432/glauk?sslmode=disable",
		MaxConns:         12,
		MinConns:         3,
		MaxConnLifetime:  time.Hour,
		MaxConnIdleTime:  30 * time.Minute,
		ConnectTimeout:   2 * time.Second,
		StatementTimeout: 2500 * time.Millisecond,
	}
}

func TestNewPoolConfig(t *testing.T) {
	cfg, err := newPoolConfig(validDBConfig())
	if err != nil {
		t.Fatalf("newPoolConfig: %v", err)
	}
	if cfg.MaxConns != 12 || cfg.MinConns != 3 {
		t.Fatalf("conexiones = %d/%d", cfg.MaxConns, cfg.MinConns)
	}
	if cfg.MaxConnLifetime != time.Hour || cfg.MaxConnIdleTime != 30*time.Minute {
		t.Fatalf("lifetimes inesperados: %v/%v", cfg.MaxConnLifetime, cfg.MaxConnIdleTime)
	}
	if cfg.HealthCheckPeriod != 30*time.Second {
		t.Fatalf("HealthCheckPeriod = %v", cfg.HealthCheckPeriod)
	}
	if cfg.ConnConfig.ConnectTimeout != 2*time.Second {
		t.Fatalf("ConnectTimeout = %v", cfg.ConnConfig.ConnectTimeout)
	}
	params := cfg.ConnConfig.RuntimeParams
	if params["application_name"] != "glauk" {
		t.Fatalf("application_name = %q", params["application_name"])
	}
	if params["statement_timeout"] != "2500" {
		t.Fatalf("statement_timeout = %q", params["statement_timeout"])
	}
}

func TestNewPoolConfigRejectsInvalidValues(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*config.DBConfig)
	}{
		{"max conns", func(cfg *config.DBConfig) { cfg.MaxConns = 0 }},
		{"min conns negativa", func(cfg *config.DBConfig) { cfg.MinConns = -1 }},
		{"min mayor que max", func(cfg *config.DBConfig) { cfg.MinConns = 13 }},
		{"lifetime", func(cfg *config.DBConfig) { cfg.MaxConnLifetime = 0 }},
		{"idle", func(cfg *config.DBConfig) { cfg.MaxConnIdleTime = -time.Second }},
		{"connect timeout", func(cfg *config.DBConfig) { cfg.ConnectTimeout = 0 }},
		{"statement timeout", func(cfg *config.DBConfig) { cfg.StatementTimeout = time.Microsecond }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validDBConfig()
			tc.mutate(&cfg)
			if _, err := newPoolConfig(cfg); err == nil {
				t.Fatal("se esperaba error de validación")
			}
		})
	}
}

func TestNewPoolConfigRejectsInvalidDSNWithoutLeaking(t *testing.T) {
	cfg := validDBConfig()
	cfg.DatabaseURL = "postgres://user:supersecret@[::1/glauk"
	_, err := newPoolConfig(cfg)
	if err == nil {
		t.Fatal("se esperaba error de DSN")
	}
	if strings.Contains(err.Error(), "supersecret") {
		t.Fatalf("el error filtró la contraseña: %v", err)
	}
}

func TestNewFailsFastWhenDatabaseIsUnreachable(t *testing.T) {
	cfg := validDBConfig()
	cfg.DatabaseURL = "postgres://user:pass@127.0.0.1:1/glauk?sslmode=disable"
	cfg.MinConns = 0
	cfg.ConnectTimeout = 200 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := time.Now()
	pool, err := New(ctx, cfg)
	elapsed := time.Since(start)
	if err == nil {
		pool.Close()
		t.Fatal("New debía fallar sin base disponible")
	}
	if elapsed > time.Second {
		t.Fatalf("fallo lento: %v", elapsed)
	}
}
