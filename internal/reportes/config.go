package reportes

import "time"

const (
	defaultImageMaxBytes     int64 = 10 * 1024 * 1024
	defaultImageMinDimension       = 200
	defaultImageMaxDimension       = 4096
	defaultImageMaxPixels    int64 = 16_000_000
	defaultIdempotencyTTL          = 10 * time.Minute
)

// Config es la porción de configuración que el dominio necesita. Se construye
// en cmd/server a partir de platform/config; el dominio no lee entorno.
type Config struct {
	ImageMaxBytes     int64
	ImageMinDimension int
	ImageMaxDimension int
	ImageMaxPixels    int64
	IdempotencyTTL    time.Duration
}

func (c Config) withDefaults() Config {
	if c.ImageMaxBytes <= 0 {
		c.ImageMaxBytes = defaultImageMaxBytes
	}
	if c.ImageMinDimension <= 0 {
		c.ImageMinDimension = defaultImageMinDimension
	}
	if c.ImageMaxDimension <= 0 {
		c.ImageMaxDimension = defaultImageMaxDimension
	}
	if c.ImageMaxPixels <= 0 {
		c.ImageMaxPixels = defaultImageMaxPixels
	}
	if c.IdempotencyTTL <= 0 {
		c.IdempotencyTTL = defaultIdempotencyTTL
	}
	return c
}
