package reportes

import (
	"strings"
	"sync"
	"time"
)

const defaultRateLimitPerMinute = 60

type tokenBucket struct {
	tokens float64
	last   time.Time
}

// RateLimiter aplica un token bucket por IP. El dominio no conoce Fiber: el
// handler entrega c.IP() y decide el código 429 y la cabecera Retry-After.
type RateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*tokenBucket
	perMinute int
	burst     float64
	now       func() time.Time
}

// NewRateLimiter crea el limitador con 60/minuto y ráfaga 10 por defecto.
func NewRateLimiter(perMinute int) *RateLimiter {
	if perMinute <= 0 {
		perMinute = defaultRateLimitPerMinute
	}
	return &RateLimiter{
		buckets: make(map[string]*tokenBucket), perMinute: perMinute,
		burst: 10, now: time.Now,
	}
}

// Allow consume un token. La ráfaga inicial permite diez envíos seguidos y
// luego se repone a razón de perMinute/60 por segundo.
func (r *RateLimiter) Allow(ip string) bool {
	if r == nil || r.perMinute <= 0 {
		return true
	}
	key := strings.TrimSpace(ip)
	if key == "" {
		key = "unknown"
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	bucket, ok := r.buckets[key]
	if !ok {
		bucket = &tokenBucket{tokens: r.burst, last: now}
		r.buckets[key] = bucket
	}
	bucket.tokens += now.Sub(bucket.last).Seconds() * float64(r.perMinute) / 60
	if bucket.tokens > r.burst {
		bucket.tokens = r.burst
	}
	bucket.last = now
	if bucket.tokens < 1 {
		return false
	}
	bucket.tokens--
	return true
}

// RetryAfter devuelve los segundos mínimos para recuperar un token.
func (r *RateLimiter) RetryAfter(ip string) int {
	if r == nil || r.perMinute <= 0 {
		return 0
	}
	key := strings.TrimSpace(ip)
	if key == "" {
		key = "unknown"
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	bucket, ok := r.buckets[key]
	if !ok || bucket.tokens >= 1 {
		return 0
	}
	missing := 1 - bucket.tokens
	seconds := missing * 60 / float64(r.perMinute)
	if seconds < 1 {
		return 1
	}
	return int(seconds)
}
