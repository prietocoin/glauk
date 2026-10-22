package config

import (
	"fmt"
	"strings"
)

// Errors aggregates failures so one startup attempt reports every problem.
type Errors []string

func (e Errors) Error() string {
	if len(e) == 0 {
		return ""
	}
	label := "errores"
	if len(e) == 1 {
		label = "error"
	}
	return fmt.Sprintf("configuración inválida (%d %s):\n  - %s",
		len(e), label, strings.Join(e, "\n  - "))
}

func (c *Config) validate() error {
	var errs Errors
	c.validateCore(&errs)
	c.validateDatabase(&errs)
	c.validateEvolution(&errs)
	c.validateHTTP(&errs)
	c.validateReports(&errs)
	c.validateHOO(&errs)
	if c.AppEnv == "production" {
		c.validateProduction(&errs)
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

func (c *Config) validateCore(errs *Errors) {
	if c.AppEnv != "development" && c.AppEnv != "staging" && c.AppEnv != "production" {
		*errs = append(*errs, "APP_ENV debe ser development, staging o production")
	}
	if c.Port < 1 || c.Port > 65535 {
		*errs = append(*errs, "PORT debe estar entre 1 y 65535")
	}
	if c.ShutdownTimeout <= 0 {
		*errs = append(*errs, "SHUTDOWN_TIMEOUT debe ser positivo")
	}
	if strings.TrimSpace(c.TimeZone) == "" {
		*errs = append(*errs, "TZ no puede estar vacío")
	}
	switch c.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		*errs = append(*errs, "LOG_LEVEL debe ser debug, info, warn o error")
	}
	switch c.Log.Format {
	case "auto", "json", "text":
	default:
		*errs = append(*errs, "LOG_FORMAT debe ser auto, json o text")
	}
}

func (c *Config) validateDatabase(errs *Errors) {
	if c.DB.Port < 1 || c.DB.Port > 65535 {
		*errs = append(*errs, "DB_PORT debe estar entre 1 y 65535")
	}
	if c.DB.MaxConns < 1 || c.DB.MinConns < 0 || c.DB.MinConns > c.DB.MaxConns {
		*errs = append(*errs, "DB_MIN_CONNS/DB_MAX_CONNS inconsistentes")
	}
	switch c.DB.SSLMode {
	case "disable", "require", "verify-ca", "verify-full":
	default:
		*errs = append(*errs, "DB_SSLMODE debe ser disable, require, verify-ca o verify-full")
	}
	if c.AppEnv != "development" && c.DB.DatabaseURL == "" && !c.DB.sourceConfigured {
		*errs = append(*errs, "DATABASE_URL es obligatoria en "+c.AppEnv+
			" (o defina DB_HOST/DB_USER/DB_NAME)")
	}
	if c.AppEnv == "production" && c.DB.DatabaseURL == "" &&
		c.DB.sourceConfigured && c.DB.Password == "" {
		*errs = append(*errs, "DB_PASSWORD es obligatoria en production al componer el DSN")
	}
	if c.DB.DatabaseURL != "" {
		if err := ValidateDSN(c.DB.DatabaseURL); err != nil {
			*errs = append(*errs, err.Error())
		}
	}
}

func (c *Config) validateEvolution(errs *Errors) {
	if c.Evolution.keyConflict {
		*errs = append(*errs, "EVOLUTION_API_KEY y AUTHENTICATION_API_KEY deben coincidir")
	}
	if (c.Evolution.BaseURL == "") != (c.Evolution.APIKey == "") {
		*errs = append(*errs, "EVOLUTION_API_URL y EVOLUTION_API_KEY deben definirse juntas")
	}
	if c.Evolution.BaseURL != "" && !validHTTPURL(c.Evolution.BaseURL) {
		*errs = append(*errs, "EVOLUTION_API_URL debe ser una URL http(s) con host")
	}
	if strings.TrimSpace(c.Evolution.Instance) == "" {
		*errs = append(*errs, "EVOLUTION_INSTANCE no puede estar vacía")
	}
	if c.Evolution.MaxRetries < 0 || c.Evolution.MaxRetries > 5 {
		*errs = append(*errs, "EVOLUTION_MAX_RETRIES debe estar entre 0 y 5")
	}
	if c.Evolution.Timeout <= 0 || c.Evolution.RetryBase <= 0 {
		*errs = append(*errs, "los timeouts de Evolution deben ser positivos")
	}
}

func (c *Config) validateHTTP(errs *Errors) {
	if c.HTTP.BodyLimit <= 0 {
		*errs = append(*errs, "BODY_LIMIT_BYTES debe ser positivo")
	}
	if c.HTTP.ReadTimeout <= 0 || c.HTTP.WriteTimeout <= 0 || c.HTTP.IdleTimeout <= 0 {
		*errs = append(*errs, "los timeouts HTTP deben ser positivos")
	}
	if len(c.HTTP.AllowedOrigins) == 0 {
		*errs = append(*errs, "CORS_ALLOWED_ORIGINS no puede estar vacío")
	}
}
