package config

import "strings"

// validateIngesta solo exige credenciales cuando el pipeline está activado.
// Así un despliegue que todavía usa remitHub sigue arrancando sin cambios.
func (c *Config) validateIngesta(errs *Errors) {
	ing := c.Ingesta
	if ing.QueueSize < 1 || ing.QueueSize > 10_000 {
		*errs = append(*errs, "INGESTA_QUEUE_SIZE debe estar entre 1 y 10000")
	}
	if ing.Workers < 1 || ing.Workers > 32 {
		*errs = append(*errs, "INGESTA_WORKERS debe estar entre 1 y 32")
	}
	if ing.MaxRetries < 0 || ing.MaxRetries > 10 {
		*errs = append(*errs, "INGESTA_MAX_RETRIES debe estar entre 0 y 10")
	}
	if ing.RetryBase <= 0 || ing.JobTimeout <= 0 {
		*errs = append(*errs, "los tiempos de INGESTA deben ser positivos")
	}
	if ing.MaxImageBytes <= 0 {
		*errs = append(*errs, "INGESTA_MAX_IMAGE_BYTES debe ser positivo")
	}
	if !ing.Enabled {
		return
	}

	if c.Evolution.BaseURL == "" {
		*errs = append(*errs, "INGESTA_ENABLED requiere EVOLUTION_API_URL")
	}
	if strings.TrimSpace(ing.R2.AccessKeyID) == "" ||
		strings.TrimSpace(ing.R2.SecretAccessKey) == "" ||
		strings.TrimSpace(ing.R2.Bucket) == "" {
		*errs = append(*errs, "INGESTA_ENABLED requiere R2_ACCESS_KEY_ID, "+
			"R2_SECRET_ACCESS_KEY y R2_BUCKET_NAME")
	}
	if strings.TrimSpace(ing.R2.Endpoint) == "" &&
		strings.TrimSpace(ing.R2.AccountID) == "" {
		*errs = append(*errs, "INGESTA_ENABLED requiere R2_ACCOUNT_ID o R2_ENDPOINT")
	}
	if endpoint := strings.TrimSpace(ing.R2.Endpoint); endpoint != "" &&
		!validHTTPURL(endpoint) {
		*errs = append(*errs, "R2_ENDPOINT debe ser una URL http(s) con host")
	}
	if domain := strings.TrimSpace(ing.R2.PublicDomain); domain != "" &&
		!validHTTPURL(domain) {
		*errs = append(*errs, "R2_PUBLIC_DOMAIN debe ser una URL http(s) con host")
	}
	if len(ing.Gemini.APIKeys) == 0 {
		*errs = append(*errs, "INGESTA_ENABLED requiere GEMINI_API_KEY o GEMINI_KEYS")
	}
	if strings.TrimSpace(ing.Gemini.Model) == "" {
		*errs = append(*errs, "GEMINI_MODEL no puede estar vacío")
	}
	if ing.Gemini.Timeout <= 0 {
		*errs = append(*errs, "GEMINI_TIMEOUT debe ser positivo")
	}
}
