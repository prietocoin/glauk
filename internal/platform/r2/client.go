// Package r2 habla con Cloudflare R2 mediante la API compatible con S3.
// Solo conoce HTTP y firmas: no decide nombres de objeto ni reglas de negocio.
package r2

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"glauk/internal/platform/config"
)

const maxErrorBody = 4 << 10
const maxObjectBytes = 32 << 20

// Client guarda credenciales y resuelve objetos dentro de un único bucket.
type Client struct {
	endpoint     string
	bucket       string
	publicDomain string
	signer       signer
	http         *http.Client
	now          func() time.Time
}

// New construye el cliente. Si no hay endpoint se deriva el de R2 con la
// cuenta; el dominio público es opcional (sin él se guarda la clave).
func New(cfg config.R2Config) *Client {
	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	if endpoint == "" && strings.TrimSpace(cfg.AccountID) != "" {
		endpoint = "https://" + strings.TrimSpace(cfg.AccountID) + ".r2.cloudflarestorage.com"
	}
	return &Client{
		endpoint:     endpoint,
		bucket:       strings.TrimSpace(cfg.Bucket),
		publicDomain: strings.TrimRight(strings.TrimSpace(cfg.PublicDomain), "/"),
		signer:       signer{accessKey: strings.TrimSpace(cfg.AccessKeyID), secretKey: strings.TrimSpace(cfg.SecretAccessKey)},
		http:         &http.Client{Timeout: 60 * time.Second},
		now:          time.Now,
	}
}

// Configured permite que el pipeline falle con un mensaje claro en vez de
// intentar una firma con credenciales vacías.
func (c *Client) Configured() bool {
	return c != nil && c.endpoint != "" && c.bucket != "" &&
		c.signer.accessKey != "" && c.signer.secretKey != ""
}

// Put sube un objeto y devuelve la URL pública configurada (o la clave).
func (c *Client) Put(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	if !c.Configured() {
		return "", fmt.Errorf("r2: credenciales o bucket incompletos")
	}
	if strings.TrimSpace(key) == "" {
		return "", fmt.Errorf("r2: la clave del objeto es obligatoria")
	}
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	resp, err := c.do(ctx, http.MethodPut, key, data, contentType)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", statusError("subir", resp)
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorBody))
	return c.PublicURL(key), nil
}

// Get descarga un objeto; se usa como respaldo cuando Evolution no entrega
// el binario pero la imagen ya estaba en R2.
func (c *Client) Get(ctx context.Context, key string) ([]byte, string, error) {
	if !c.Configured() {
		return nil, "", fmt.Errorf("r2: credenciales o bucket incompletos")
	}
	resp, err := c.do(ctx, http.MethodGet, key, nil, "")
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", statusError("descargar", resp)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxObjectBytes))
	if err != nil {
		return nil, "", fmt.Errorf("r2: leer objeto: %w", err)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	return data, contentType, nil
}

// PublicURL conserva la misma regla del servicio Node: dominio público si
// existe; en su defecto, la clave para que el panel la resuelva.
func (c *Client) PublicURL(key string) string {
	clean := strings.TrimLeft(key, "/")
	if c.publicDomain != "" {
		return c.publicDomain + "/" + clean
	}
	return clean
}

func (c *Client) do(ctx context.Context, method, key string, data []byte,
	contentType string) (*http.Response, error) {
	target, err := c.objectURL(key)
	if err != nil {
		return nil, err
	}
	body := io.Reader(nil)
	if data != nil {
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return nil, fmt.Errorf("r2: construir petición: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	c.signer.sign(req, data, c.now())
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("r2: %s objeto: %w", method, err)
	}
	return resp, nil
}

func (c *Client) objectURL(key string) (*url.URL, error) {
	base, err := url.Parse(c.endpoint)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("r2: endpoint inválido: %q", c.endpoint)
	}
	copied := *base
	prefix := strings.TrimRight(base.Path, "/")
	copied.Path = prefix + "/" + c.bucket + "/" + strings.TrimLeft(key, "/")
	return &copied, nil
}

func statusError(action string, resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	return fmt.Errorf("r2: %s objeto: estado %d: %s", action, resp.StatusCode,
		strings.TrimSpace(string(raw)))
}
