package config

import "time"

const (
	defaultIngestaWorkers    = 5
	defaultIngestaQueue      = 100
	defaultIngestaRetries    = 3
	defaultIngestaRetryBase  = 500 * time.Millisecond
	defaultIngestaJobTimeout = 2 * time.Minute
	defaultIngestaImageBytes = 20 << 20
	defaultGeminiModel       = "gemini-1.5-flash"
	defaultGeminiTimeout     = 45 * time.Second
	defaultR2Bucket          = "remesas-img"
)

// IngestaConfig describe el pipeline nativo que reemplaza a remitHub y a
// glaukapi. Deshabilitado por defecto: activarlo exige credenciales de R2 y
// Gemini, y el operador debe hacerlo de forma explícita.
type IngestaConfig struct {
	Enabled       bool
	QueueSize     int
	Workers       int
	MaxRetries    int
	RetryBase     time.Duration
	JobTimeout    time.Duration
	MaxImageBytes int64
	EscribirFinal bool
	R2            R2Config
	Gemini        GeminiConfig
}

// R2Config agrupa las credenciales del almacenamiento compatible con S3.
type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicDomain    string
	Endpoint        string
}

// GeminiConfig conserva la lista de claves para rotarlas ante 429/503 y el
// modelo preferido; el cliente añade su propia cascada de respaldo.
type GeminiConfig struct {
	APIKey  string
	APIKeys []string
	Model   string
	Timeout time.Duration
}

func buildIngesta(l lookupFunc, errs *Errors) IngestaConfig {
	cfg := IngestaConfig{
		QueueSize:     defaultIngestaQueue,
		Workers:       defaultIngestaWorkers,
		MaxRetries:    defaultIngestaRetries,
		RetryBase:     defaultIngestaRetryBase,
		JobTimeout:    defaultIngestaJobTimeout,
		MaxImageBytes: defaultIngestaImageBytes,
		R2: R2Config{
			AccountID:       envString(l, "R2_ACCOUNT_ID", ""),
			AccessKeyID:     envString(l, "R2_ACCESS_KEY_ID", ""),
			SecretAccessKey: envString(l, "R2_SECRET_ACCESS_KEY", ""),
			Bucket:          envString(l, "R2_BUCKET_NAME", defaultR2Bucket),
			PublicDomain:    envString(l, "R2_PUBLIC_DOMAIN", ""),
			Endpoint:        envString(l, "R2_ENDPOINT", ""),
		},
		Gemini: GeminiConfig{
			APIKey:  envString(l, "GEMINI_API_KEY", ""),
			APIKeys: envCSV(l, "GEMINI_KEYS", nil),
			Model:   envString(l, "GEMINI_MODEL", defaultGeminiModel),
			Timeout: defaultGeminiTimeout,
		},
	}

	readBool(l, "INGESTA_ENABLED", false, &cfg.Enabled, errs)
	readInt(l, "INGESTA_QUEUE_SIZE", defaultIngestaQueue, &cfg.QueueSize, errs)
	readInt(l, "INGESTA_WORKERS", defaultIngestaWorkers, &cfg.Workers, errs)
	readInt(l, "INGESTA_MAX_RETRIES", defaultIngestaRetries, &cfg.MaxRetries, errs)
	readDuration(l, "INGESTA_RETRY_BASE", defaultIngestaRetryBase, &cfg.RetryBase, errs)
	readDuration(l, "INGESTA_JOB_TIMEOUT", defaultIngestaJobTimeout, &cfg.JobTimeout, errs)
	readBytes(l, "INGESTA_MAX_IMAGE_BYTES", defaultIngestaImageBytes, &cfg.MaxImageBytes, errs)
	readBool(l, "INGESTA_ESCRIBIR_FINAL", false, &cfg.EscribirFinal, errs)
	readDuration(l, "GEMINI_TIMEOUT", defaultGeminiTimeout, &cfg.Gemini.Timeout, errs)

	return cfg
}

// addKey incorpora una clave sin duplicar y sin conservar espacios.
func (g *GeminiConfig) addKey(key string) {
	for _, current := range g.APIKeys {
		if current == key {
			return
		}
	}
	g.APIKeys = append(g.APIKeys, key)
}
