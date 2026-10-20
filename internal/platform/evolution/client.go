package evolution

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"glauk/internal/platform/config"
)

const maxResponseBytes = 1 << 20

// Client speaks only HTTP to Evolution. It has no JIDs, SQL or domain rules.
type Client struct {
	baseURL    string
	apiKey     string
	instance   string
	http       *http.Client
	maxRetries int
	retryBase  time.Duration
	sleep      func(context.Context, time.Duration) error
	randFloat  func() float64
}

// New builds a stateless client; the API key stays in memory for the apikey header.
func New(cfg config.EvolutionConfig) *Client {
	maxRetries := cfg.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	return &Client{
		baseURL:    strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		apiKey:     strings.TrimSpace(cfg.APIKey),
		instance:   strings.TrimSpace(cfg.Instance),
		http:       &http.Client{Timeout: cfg.Timeout},
		maxRetries: maxRetries,
		retryBase:  cfg.RetryBase,
		sleep:      waitContext,
		randFloat:  rand.Float64,
	}
}

// Simulated reports whether EVOLUTION_API_URL activated the offline mode.
func (c *Client) Simulated() bool { return c.baseURL == "" }

// send serializa una vez y reintenta solo fallos transitorios.
func (c *Client) send(ctx context.Context, path string, payload any) (Receipt, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return Receipt{}, fmt.Errorf("serializar payload de Evolution: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return Receipt{}, err
		}
		if attempt > 0 {
			if err := c.sleep(ctx, c.backoff(attempt, lastErr)); err != nil {
				return Receipt{}, err
			}
		}
		receipt, retry, err := c.once(ctx, path, body)
		if err == nil {
			return receipt, nil
		}
		lastErr = err
		if !retry || ctx.Err() != nil {
			break
		}
	}
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}
	return Receipt{}, lastErr
}

func (c *Client) once(ctx context.Context, path string, body []byte) (Receipt, bool, error) {
	endpoint := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return Receipt{}, false, fmt.Errorf("construir petición a Evolution: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Receipt{}, false, ctxErr
		}
		return Receipt{}, true, fmt.Errorf("enviar a Evolution: %w", err)
	}
	defer resp.Body.Close()

	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if readErr != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Receipt{}, false, ctxErr
		}
		return Receipt{}, true, fmt.Errorf("leer respuesta de Evolution: %w", readErr)
	}
	if c.apiKey != "" {
		raw = bytes.ReplaceAll(raw, []byte(c.apiKey), []byte("[REDACTED]"))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := classify(resp.StatusCode, raw, resp.Header.Get("Retry-After"))
		return Receipt{}, apiErr.Retryable, apiErr
	}
	return parseReceipt(raw), false, nil
}

func (c *Client) path(op string) string {
	return "/message/" + op + "/" + url.PathEscape(c.instance)
}
