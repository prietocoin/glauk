package config

import (
	"net/url"
	"strings"
)

func (c *Config) validateReports(errs *Errors) {
	if c.Report.ImageMaxBytes <= 0 {
		*errs = append(*errs, "REPORT_IMAGE_MAX_BYTES debe ser positivo")
	}
	if c.Report.ImageMinDim < 1 || c.Report.ImageMaxDim < c.Report.ImageMinDim {
		*errs = append(*errs, "las dimensiones de imagen de reportes son inconsistentes")
	}
	if c.Report.ImageMaxPixels <= 0 {
		*errs = append(*errs, "REPORT_IMAGE_MAX_PIXELS debe ser positivo")
	}
	if c.Report.IdempotencyTTL <= 0 {
		*errs = append(*errs, "REPORT_IDEMPOTENCY_TTL debe ser positivo")
	}
	if c.Report.RatePerMinute < 0 {
		*errs = append(*errs, "REPORT_RATE_LIMIT_PER_MINUTE no puede ser negativo")
	}
}

func (c *Config) validateHOO(errs *Errors) {
	if c.HOO.URL != "" && !validHTTPURL(c.HOO.URL) {
		*errs = append(*errs, "HOO_GLOBAL_RATES_URL debe ser una URL http(s) con host")
	}
	if c.HOO.Timeout <= 0 {
		*errs = append(*errs, "HOO_TIMEOUT debe ser positivo")
	}
}

func (c *Config) validateProduction(errs *Errors) {
	mode := productionSSLMode(c.DB)
	if (mode == "" || mode == "disable") && !c.DB.AllowInsecure {
		*errs = append(*errs, "production exige sslmode=require o superior (o DB_ALLOW_INSECURE=true)")
	}
	if c.Evolution.BaseURL == "" {
		*errs = append(*errs, "EVOLUTION_API_URL es obligatoria en production")
	}
	if len(c.HTTP.AllowedOrigins) == 1 && c.HTTP.AllowedOrigins[0] == "*" {
		*errs = append(*errs, `CORS_ALLOWED_ORIGINS="*" no está permitido en production`)
	}
	if c.EnablePprof {
		*errs = append(*errs, "ENABLE_PPROF no está permitido en production")
	}
}

func productionSSLMode(db DBConfig) string {
	if db.DatabaseURL == "" {
		return db.SSLMode
	}
	u, err := url.Parse(db.DatabaseURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Query().Get("sslmode"))
}

func validHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}
