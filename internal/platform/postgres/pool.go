package postgres

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"glauk/internal/platform/config"
)

const healthCheckPeriod = 30 * time.Second

// New builds the single pool owned by cmd/server and verifies connectivity
// before any request can use it. A failed ping closes the pool immediately.
func New(ctx context.Context, cfg config.DBConfig) (*pgxpool.Pool, error) {
	poolCfg, err := newPoolConfig(cfg)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("crear pool de PostgreSQL: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("verificar conexión a PostgreSQL: %w", err)
	}
	return pool, nil
}

func newPoolConfig(cfg config.DBConfig) (*pgxpool.Config, error) {
	if err := validatePoolConfig(cfg); err != nil {
		return nil, err
	}
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parsear configuración de PostgreSQL: DSN inválido")
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = healthCheckPeriod
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	if poolCfg.ConnConfig.RuntimeParams == nil {
		poolCfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	poolCfg.ConnConfig.RuntimeParams["application_name"] = "glauk"
	poolCfg.ConnConfig.RuntimeParams["statement_timeout"] =
		strconv.FormatInt(cfg.StatementTimeout.Milliseconds(), 10)
	return poolCfg, nil
}

func validatePoolConfig(cfg config.DBConfig) error {
	switch {
	case cfg.MaxConns < 1:
		return fmt.Errorf("DB_MAX_CONNS debe ser positivo")
	case cfg.MinConns < 0 || cfg.MinConns > cfg.MaxConns:
		return fmt.Errorf("DB_MIN_CONNS debe estar entre 0 y DB_MAX_CONNS")
	case cfg.MaxConnLifetime <= 0:
		return fmt.Errorf("DB_CONN_MAX_LIFETIME debe ser positivo")
	case cfg.MaxConnIdleTime <= 0:
		return fmt.Errorf("DB_CONN_MAX_IDLE_TIME debe ser positivo")
	case cfg.ConnectTimeout <= 0:
		return fmt.Errorf("DB_CONNECT_TIMEOUT debe ser positivo")
	case cfg.StatementTimeout < time.Millisecond:
		return fmt.Errorf("DB_STATEMENT_TIMEOUT debe ser al menos 1ms")
	}
	return nil
}
