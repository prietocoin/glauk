package config

import (
	"os"
	"time"
)

// Config is immutable once built. It is injected by pointer and no package
// outside platform/config mutates it.
type Config struct {
	AppEnv          string
	Port            int
	ShutdownTimeout time.Duration
	TimeZone        string
	EnablePprof     bool
	DB              DBConfig
	Evolution       EvolutionConfig
	HTTP            HTTPConfig
	Report          ReportConfig
	Admin           AdminConfig
	HOO             HOOConfig
	Tenants         TenantConfig
	Ingesta         IngestaConfig
	Log             LogConfig
}
type DBConfig struct {
	DatabaseURL      string
	Host             string
	Port             int
	User             string
	Password         string
	Name             string
	SSLMode          string
	AllowInsecure    bool
	MaxConns         int32
	MinConns         int32
	MaxConnLifetime  time.Duration
	MaxConnIdleTime  time.Duration
	ConnectTimeout   time.Duration
	StatementTimeout time.Duration
	sourceConfigured bool
}

type EvolutionConfig struct {
	BaseURL     string
	APIKey      string
	Instance    string
	Timeout     time.Duration
	MaxRetries  int
	RetryBase   time.Duration
	keyConflict bool
}

type HTTPConfig struct {
	BodyLimit      int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	AllowedOrigins []string
	SecurityAPIKey string
}

type ReportConfig struct {
	ImageMaxBytes  int64
	ImageMinDim    int
	ImageMaxDim    int
	ImageMaxPixels int64
	IdempotencyTTL time.Duration
	RatePerMinute  int
}

type AdminConfig struct{ APIKey string }

type HOOConfig struct {
	URL     string
	Timeout time.Duration
}

type TenantConfig struct {
	JIDGrupoPrueba string
}

type LogConfig struct {
	Level  string
	Format string
}

// Load reads the environment once and returns all failures together.
func Load() (*Config, error) {
	appEnv := envString(os.LookupEnv, "APP_ENV", "development")
	cfg, errs := build(processEnv(appEnv), appEnv)
	if err := cfg.validate(); err != nil {
		if list, ok := err.(Errors); ok {
			errs = append(errs, list...)
		} else {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return nil, errs
	}
	return cfg, nil
}
