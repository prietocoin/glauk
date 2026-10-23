package config

import (
	"strings"
	"time"
)

const (
	defaultPort         = 3000
	defaultBodyLimit    = 20 << 20
	defaultImageBytes   = 10 << 20
	defaultImagePixels  = 16_000_000
	defaultShutdownTime = 15 * time.Second
)

// build reads every variable through lookup and accumulates parse failures.
// It never inspects the process environment directly.
func build(l lookupFunc, appEnv string) (*Config, Errors) {
	var errs Errors
	cfg := &Config{
		AppEnv:          appEnv,
		Port:            defaultPort,
		ShutdownTimeout: defaultShutdownTime,
		TimeZone:        envString(l, "TZ", "UTC"),
		Log: LogConfig{
			Level:  envString(l, "LOG_LEVEL", "info"),
			Format: envString(l, "LOG_FORMAT", "auto"),
		},
		DB: DBConfig{
			DatabaseURL:      envString(l, "DATABASE_URL", ""),
			Host:             envString(l, "DB_HOST", "localhost"),
			Port:             5432,
			User:             envString(l, "DB_USER", "postgres"),
			Password:         envString(l, "DB_PASSWORD", ""),
			Name:             envString(l, "DB_NAME", "automat"),
			SSLMode:          strings.ToLower(envString(l, "DB_SSLMODE", "disable")),
			MaxConns:         20,
			MinConns:         2,
			MaxConnLifetime:  time.Hour,
			MaxConnIdleTime:  30 * time.Minute,
			ConnectTimeout:   10 * time.Second,
			StatementTimeout: 10 * time.Second,
		},
		Evolution: EvolutionConfig{
			BaseURL:    strings.TrimRight(envString(l, "EVOLUTION_API_URL", ""), "/"),
			Instance:   envString(l, "EVOLUTION_INSTANCE", "Glauk"),
			Timeout:    25 * time.Second,
			MaxRetries: 2,
			RetryBase:  300 * time.Millisecond,
		},
		HTTP: HTTPConfig{
			BodyLimit:    defaultBodyLimit,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 90 * time.Second,
			IdleTimeout:  60 * time.Second,
			AllowedOrigins: envCSV(l, "CORS_ALLOWED_ORIGINS",
				[]string{"https://automat-glauk-ui.fyi6ur.easypanel.host", "http://localhost:*"}),
			SecurityAPIKey: envString(l, "SECURITY_API_KEY", ""),
		},
		Report: ReportConfig{
			ImageMaxBytes:  defaultImageBytes,
			ImageMinDim:    200,
			ImageMaxDim:    4096,
			ImageMaxPixels: defaultImagePixels,
			IdempotencyTTL: 10 * time.Minute,
			RatePerMinute:  60,
		},
		Admin: AdminConfig{APIKey: envString(l, "ADMIN_API_KEY", "")},
		HOO: HOOConfig{
			URL:     envString(l, "HOO_GLOBAL_RATES_URL", "https://hoo.jairokov.com/global-rates"),
			Timeout: 10 * time.Second,
		},
		Tenants: TenantConfig{
			JIDGrupoPrueba: envString(l, "JID_GRUPO_PRUEBA", ""),
		},
	}

	canonicalKey := envString(l, "EVOLUTION_API_KEY", "")
	legacyKey := envString(l, "AUTHENTICATION_API_KEY", "")
	cfg.Evolution.APIKey = canonicalKey
	if cfg.Evolution.APIKey == "" {
		cfg.Evolution.APIKey = legacyKey
	}
	cfg.Evolution.keyConflict = canonicalKey != "" && legacyKey != "" && canonicalKey != legacyKey

	readInt(l, "PORT", defaultPort, &cfg.Port, &errs)
	readDuration(l, "SHUTDOWN_TIMEOUT", defaultShutdownTime, &cfg.ShutdownTimeout, &errs)
	readBool(l, "ENABLE_PPROF", false, &cfg.EnablePprof, &errs)

	readInt(l, "DB_PORT", 5432, &cfg.DB.Port, &errs)
	readBool(l, "DB_ALLOW_INSECURE", false, &cfg.DB.AllowInsecure, &errs)
	readInt32(l, "DB_MAX_CONNS", 20, &cfg.DB.MaxConns, &errs)
	readInt32(l, "DB_MIN_CONNS", 2, &cfg.DB.MinConns, &errs)
	readDuration(l, "DB_CONN_MAX_LIFETIME", time.Hour, &cfg.DB.MaxConnLifetime, &errs)
	readDuration(l, "DB_CONN_MAX_IDLE_TIME", 30*time.Minute, &cfg.DB.MaxConnIdleTime, &errs)
	readDuration(l, "DB_CONNECT_TIMEOUT", 10*time.Second, &cfg.DB.ConnectTimeout, &errs)
	readDuration(l, "DB_STATEMENT_TIMEOUT", 10*time.Second, &cfg.DB.StatementTimeout, &errs)

	readDuration(l, "EVOLUTION_TIMEOUT", 25*time.Second, &cfg.Evolution.Timeout, &errs)
	readInt(l, "EVOLUTION_MAX_RETRIES", 2, &cfg.Evolution.MaxRetries, &errs)
	readDuration(l, "EVOLUTION_RETRY_BASE", 300*time.Millisecond, &cfg.Evolution.RetryBase, &errs)

	readDuration(l, "HTTP_READ_TIMEOUT", 15*time.Second, &cfg.HTTP.ReadTimeout, &errs)
	readDuration(l, "HTTP_WRITE_TIMEOUT", 90*time.Second, &cfg.HTTP.WriteTimeout, &errs)
	readDuration(l, "HTTP_IDLE_TIMEOUT", 60*time.Second, &cfg.HTTP.IdleTimeout, &errs)
	readBytesInt(l, "BODY_LIMIT_BYTES", defaultBodyLimit, &cfg.HTTP.BodyLimit, &errs)

	readBytes(l, "REPORT_IMAGE_MAX_BYTES", defaultImageBytes, &cfg.Report.ImageMaxBytes, &errs)
	readInt(l, "REPORT_IMAGE_MIN_DIMENSION", 200, &cfg.Report.ImageMinDim, &errs)
	readInt(l, "REPORT_IMAGE_MAX_DIMENSION", 4096, &cfg.Report.ImageMaxDim, &errs)
	readBytes(l, "REPORT_IMAGE_MAX_PIXELS", defaultImagePixels, &cfg.Report.ImageMaxPixels, &errs)
	readDuration(l, "REPORT_IDEMPOTENCY_TTL", 10*time.Minute, &cfg.Report.IdempotencyTTL, &errs)
	readInt(l, "REPORT_RATE_LIMIT_PER_MINUTE", 60, &cfg.Report.RatePerMinute, &errs)

	readDuration(l, "HOO_TIMEOUT", 10*time.Second, &cfg.HOO.Timeout, &errs)
	cfg.Ingesta = buildIngesta(l, &errs)
	if cfg.Ingesta.Gemini.APIKey != "" {
		cfg.Ingesta.Gemini.addKey(cfg.Ingesta.Gemini.APIKey)
	}
	cfg.DB.sourceConfigured = hasDBConfig(l)
	return cfg, errs
}

func hasDBConfig(l lookupFunc) bool {
	if value, ok := l("DATABASE_URL"); ok && strings.TrimSpace(value) != "" {
		return true
	}
	return envString(l, "DB_HOST", "") != "" &&
		envString(l, "DB_USER", "") != "" &&
		envString(l, "DB_NAME", "") != ""
}
