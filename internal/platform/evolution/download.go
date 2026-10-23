package evolution

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ErrMediaUnavailable indica que la instancia no devolvió el binario. No es un
// fallo transitorio del cliente: el pipeline decide si reintenta el mensaje.
var ErrMediaUnavailable = errors.New("evolution: media no disponible")

const maxMediaResponseBytes = 32 << 20

type mediaResponse struct {
	Base64   string `json:"base64"`
	Media    string `json:"media"`
	Mimetype string `json:"mimetype"`
}

// DownloadMedia pide a Evolution el binario de una imagen ya recibida.
//
// Se prueban las dos rutas conocidas (message y chat) porque la instancia
// antigua y la nueva no exponen el mismo prefijo. Un 404 pasa a la siguiente;
// cualquier otro estado se devuelve para que el worker aplique su backoff.
func (c *Client) DownloadMedia(ctx context.Context, instance string,
	key, message json.RawMessage) ([]byte, string, error) {
	if c.Simulated() {
		return nil, "", ErrMediaUnavailable
	}
	target := firstNonEmpty(instance, c.instance)
	if strings.TrimSpace(target) == "" {
		return nil, "", errors.New("evolution: instancia vacía para descargar media")
	}
	if len(key) == 0 || len(message) == 0 {
		return nil, "", errors.New("evolution: key y message son obligatorios")
	}

	payload, err := json.Marshal(map[string]any{
		"message": map[string]any{
			"key":     json.RawMessage(key),
			"message": json.RawMessage(message),
		},
		"convertToMp4": false,
	})
	if err != nil {
		return nil, "", fmt.Errorf("serializar media de Evolution: %w", err)
	}

	paths := []string{
		"/message/getBase64FromMediaMessage/" + url.PathEscape(target),
		"/chat/getBase64FromMediaMessage/" + url.PathEscape(target),
	}
	var lastErr error
	for _, path := range paths {
		data, mime, retry, err := c.downloadOnce(ctx, path, payload)
		if err == nil {
			return data, mime, nil
		}
		lastErr = err
		if !retry {
			return nil, "", err
		}
	}
	return nil, "", lastErr
}

func (c *Client) downloadOnce(ctx context.Context, path string,
	body []byte) ([]byte, string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path,
		bytes.NewReader(body))
	if err != nil {
		return nil, "", false, fmt.Errorf("construir descarga de Evolution: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, "", false, ctxErr
		}
		return nil, "", true, fmt.Errorf("descargar media de Evolution: %w", err)
	}
	defer resp.Body.Close()

	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, maxMediaResponseBytes))
	if readErr != nil {
		return nil, "", true, fmt.Errorf("leer media de Evolution: %w", readErr)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, "", true, ErrMediaUnavailable
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := classify(resp.StatusCode, raw, resp.Header.Get("Retry-After"))
		return nil, "", apiErr.Retryable, apiErr
	}

	var parsed mediaResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, "", false, fmt.Errorf("decodificar media de Evolution: %w", err)
	}
	encoded := firstNonEmpty(parsed.Base64, parsed.Media)
	if encoded == "" {
		return nil, "", false, ErrMediaUnavailable
	}
	if index := strings.Index(encoded, "base64,"); index >= 0 {
		encoded = encoded[index+len("base64,"):]
	}
	decoded, err := decodeBase64(encoded)
	if err != nil {
		return nil, "", false, fmt.Errorf("decodificar base64 de Evolution: %w", err)
	}
	return decoded, firstNonEmpty(parsed.Mimetype, "image/jpeg"), false, nil
}

func decodeBase64(value string) ([]byte, error) {
	clean := strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r', '\t', ' ':
			return -1
		}
		return r
	}, value)
	if data, err := base64.StdEncoding.DecodeString(clean); err == nil {
		return data, nil
	}
	return base64.RawStdEncoding.DecodeString(clean)
}
