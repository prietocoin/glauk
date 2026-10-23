package config

import (
	"strings"
	"testing"
	"time"
)

func TestIngestaDefaults(t *testing.T) {
	cfg, errs := loadFrom(map[string]string{}, "development")
	if len(errs) != 0 {
		t.Fatalf("errores inesperados: %v", errs)
	}
	ing := cfg.Ingesta
	if ing.Enabled {
		t.Fatal("la ingesta debe estar deshabilitada por defecto")
	}
	if ing.QueueSize != 100 || ing.Workers != 5 || ing.MaxRetries != 3 {
		t.Fatalf("cola=%d workers=%d reintentos=%d", ing.QueueSize, ing.Workers, ing.MaxRetries)
	}
	if ing.RetryBase != 500*time.Millisecond || ing.JobTimeout != 2*time.Minute {
		t.Fatalf("tiempos = %v / %v", ing.RetryBase, ing.JobTimeout)
	}
	if ing.R2.Bucket != "remesas-img" || ing.Gemini.Model != "gemini-1.5-flash" {
		t.Fatalf("bucket=%s modelo=%s", ing.R2.Bucket, ing.Gemini.Model)
	}
}

func TestIngestaEnabledExigeCredenciales(t *testing.T) {
	_, errs := loadFrom(map[string]string{"INGESTA_ENABLED": "true"}, "development")
	text := errs.Error()
	for _, want := range []string{
		"INGESTA_ENABLED requiere EVOLUTION_API_URL",
		"R2_ACCESS_KEY_ID",
		"R2_ACCOUNT_ID o R2_ENDPOINT",
		"GEMINI_API_KEY o GEMINI_KEYS",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("falta %q en:\n%s", want, text)
		}
	}
}

func TestIngestaCompletaYClavesRotativas(t *testing.T) {
	cfg, errs := loadFrom(map[string]string{
		"INGESTA_ENABLED":      "true",
		"EVOLUTION_API_URL":    "https://evo.example",
		"EVOLUTION_API_KEY":    "secret",
		"R2_ACCOUNT_ID":        "acc",
		"R2_ACCESS_KEY_ID":     "ak",
		"R2_SECRET_ACCESS_KEY": "sk",
		"R2_BUCKET_NAME":       "remesas-img",
		"GEMINI_KEYS":          "k1,k2",
		"GEMINI_API_KEY":       "k0",
	}, "development")
	if len(errs) != 0 {
		t.Fatalf("errores inesperados: %v", errs)
	}
	if !cfg.Ingesta.Enabled {
		t.Fatal("debería estar habilitada")
	}
	if len(cfg.Ingesta.Gemini.APIKeys) != 3 {
		t.Fatalf("claves = %v", cfg.Ingesta.Gemini.APIKeys)
	}
	if cfg.Ingesta.Gemini.APIKeys[0] != "k1" || cfg.Ingesta.Gemini.APIKeys[2] != "k0" {
		t.Fatalf("orden de claves = %v", cfg.Ingesta.Gemini.APIKeys)
	}
}

func TestIngestaRangosYEndpoint(t *testing.T) {
	_, errs := loadFrom(map[string]string{
		"INGESTA_ENABLED":      "true",
		"EVOLUTION_API_URL":    "https://evo.example",
		"EVOLUTION_API_KEY":    "secret",
		"R2_ACCESS_KEY_ID":     "ak",
		"R2_SECRET_ACCESS_KEY": "sk",
		"R2_BUCKET_NAME":       "remesas-img",
		"R2_ENDPOINT":          "ftp://r2.example",
		"GEMINI_API_KEY":       "k0",
		"INGESTA_WORKERS":      "0",
		"INGESTA_QUEUE_SIZE":   "0",
		"INGESTA_MAX_RETRIES":  "11",
	}, "development")
	text := errs.Error()
	for _, want := range []string{
		"INGESTA_WORKERS debe estar entre 1 y 32",
		"INGESTA_QUEUE_SIZE debe estar entre 1 y 10000",
		"INGESTA_MAX_RETRIES debe estar entre 0 y 10",
		"R2_ENDPOINT debe ser una URL http(s) con host",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("falta %q en:\n%s", want, text)
		}
	}
}
