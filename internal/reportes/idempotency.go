package reportes

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

const idempotencyMaxEntries = 1000

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)

type idempotencyEntry struct {
	result     Resultado
	expires    time.Time
	inProgress bool
}

// IdempotencyStore reserva claves antes de enviar para que un doble clic no
// produzca dos mensajes. Es por instancia y no usa Redis ni PostgreSQL.
type IdempotencyStore struct {
	mu      sync.Mutex
	entries map[string]idempotencyEntry
	ttl     time.Duration
	max     int
	now     func() time.Time
}

// NewIdempotencyStore crea el almacén con TTL y límite de entradas.
func NewIdempotencyStore(ttl time.Duration) *IdempotencyStore {
	if ttl <= 0 {
		ttl = defaultIdempotencyTTL
	}
	return &IdempotencyStore{
		entries: make(map[string]idempotencyEntry),
		ttl:     ttl,
		max:     idempotencyMaxEntries,
		now:     time.Now,
	}
}

// Key usa la cabecera si es válida o deriva una clave estable del contenido.
func (s *IdempotencyStore) Key(explicit, destino, caption, base64 string) string {
	explicit = strings.TrimSpace(explicit)
	if idempotencyKeyPattern.MatchString(explicit) {
		return explicit
	}
	sum := sha256.Sum256([]byte(destino + "\x00" + caption + "\x00" + base64))
	return fmt.Sprintf("%x", sum)
}

// Begin reserva la clave. Si ya existe un resultado vigente lo devuelve para
// que el handler responda duplicate:true; si está en curso responde 409.
func (s *IdempotencyStore) Begin(key string) (*Resultado, bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, false, ErrEnvioEnCurso
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	s.purgeLocked(now)
	if entry, ok := s.entries[key]; ok {
		if entry.inProgress {
			return nil, false, ErrEnvioEnCurso
		}
		if now.Before(entry.expires) {
			result := entry.result
			return &result, false, nil
		}
		delete(s.entries, key)
	}
	if len(s.entries) >= s.max {
		return nil, false, ErrIdempotenciaLlena
	}
	s.entries[key] = idempotencyEntry{inProgress: true, expires: now.Add(s.ttl)}
	return nil, true, nil
}

// Commit guarda el resultado exitoso y renueva el TTL de la clave.
func (s *IdempotencyStore) Commit(key string, result Resultado) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	s.entries[key] = idempotencyEntry{result: result, expires: now.Add(s.ttl)}
}

// Abort libera una reserva fallida para permitir el reintento.
func (s *IdempotencyStore) Abort(key string) {
	key = strings.TrimSpace(key)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, key)
}

func (s *IdempotencyStore) purgeLocked(now time.Time) {
	for key, entry := range s.entries {
		if !entry.inProgress && !now.Before(entry.expires) {
			delete(s.entries, key)
		}
	}
}
