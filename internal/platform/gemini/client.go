// Package gemini extrae los campos de un comprobante con la API REST de
// Google. No conoce PostgreSQL ni el pipeline: recibe bytes y devuelve datos.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"glauk/internal/platform/config"
)

const baseURL = "https://generativelanguage.googleapis.com/v1beta/models/"
const maxResponseBytes = 4 << 20

var fallbackModels = []string{
	"gemini-2.0-flash",
	"gemini-2.0-flash-lite",
	"gemini-1.5-flash",
	"gemini-1.5-flash-8b",
}

// Client rota claves y modelos ante 429/503 sin exponer esas claves en logs.
type Client struct {
	keys   []string
	models []string
	http   *http.Client
	next   int
	sleep  func(context.Context, time.Duration) error
}

// New construye el cliente con la cascada configurada más los respaldos.
func New(cfg config.GeminiConfig) *Client {
	models := []string{strings.TrimSpace(cfg.Model)}
	models = append(models, fallbackModels...)
	return &Client{
		keys:   append([]string(nil), cfg.APIKeys...),
		models: uniqueNonEmpty(models),
		http:   &http.Client{Timeout: cfg.Timeout},
		sleep:  waitContext,
	}
}

// Configured indica si hay al menos una clave utilizable.
func (c *Client) Configured() bool { return c != nil && len(c.keys) > 0 }

// Extract envía la imagen y devuelve los campos normalizados del comprobante.
func (c *Client) Extract(ctx context.Context, prompt string, image []byte,
	mimeType string) (Result, error) {
	if !c.Configured() {
		return Result{}, fmt.Errorf("gemini: no hay claves configuradas")
	}
	if len(image) == 0 {
		return Result{}, fmt.Errorf("gemini: la imagen está vacía")
	}
	if strings.TrimSpace(mimeType) == "" {
		mimeType = http.DetectContentType(image)
	}

	body, err := json.Marshal(request{
		Contents: []content{{
			Parts: []part{
				{Text: prompt},
				{InlineData: &inlineData{MimeType: mimeType, Data: image}},
			},
		}},
		GenerationConfig: generation{MimeType: "application/json", Temperature: 0.1},
	})
	if err != nil {
		return Result{}, fmt.Errorf("gemini: serializar petición: %w", err)
	}

	attempts := len(c.keys)
	if attempts > 3 {
		attempts = 3
	}
	var lastErr error
	for _, model := range c.models {
		for attempt := 0; attempt < attempts; attempt++ {
			if err := ctx.Err(); err != nil {
				return Result{}, err
			}
			key := c.takeKey()
			result, retry, err := c.once(ctx, model, key, body)
			if err == nil {
				return result, nil
			}
			lastErr = err
			if !retry {
				break
			}
			if sleepErr := c.sleep(ctx, time.Duration(150*(attempt+1))*time.Millisecond); sleepErr != nil {
				return Result{}, sleepErr
			}
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("gemini: sin respuesta")
	}
	return Result{}, fmt.Errorf("gemini: fallaron todos los modelos: %w", lastErr)
}

func (c *Client) takeKey() string {
	key := c.keys[c.next%len(c.keys)]
	c.next = (c.next + 1) % len(c.keys)
	return key
}

func (c *Client) once(ctx context.Context, model, key string, body []byte) (Result, bool, error) {
	endpoint := baseURL + url.PathEscape(model) + ":generateContent?key=" + url.QueryEscape(key)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return Result{}, false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Result{}, false, ctxErr
		}
		return Result{}, true, err
	}
	defer resp.Body.Close()

	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if readErr != nil {
		return Result{}, true, fmt.Errorf("leer respuesta: %w", readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, retryable(resp.StatusCode),
			fmt.Errorf("estado %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}

	var parsed response
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Result{}, true, fmt.Errorf("decodificar respuesta: %w", err)
	}
	text := parsed.text()
	if text == "" {
		return Result{}, true, fmt.Errorf("respuesta sin contenido")
	}
	fields, err := ParseJSON(text)
	if err != nil {
		return Result{}, true, err
	}
	return fields.Normalize(), false, nil
}

func retryable(status int) bool {
	switch status {
	case http.StatusTooManyRequests, http.StatusInternalServerError,
		http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

func truncate(value string, limit int) string {
	clean := strings.TrimSpace(value)
	if len(clean) <= limit {
		return clean
	}
	return clean[:limit] + "..."
}

func uniqueNonEmpty(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

func waitContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
