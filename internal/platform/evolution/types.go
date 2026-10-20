// Package evolution is the only outbound adapter for Evolution API.
// It knows payloads, HTTP and retries; it knows nothing about JIDs or SQL.
package evolution

import (
	"encoding/json"
	"fmt"
)

// MediaInput es el contrato exacto de Evolution para sendMedia.
type MediaInput struct {
	// Number is the destination as supplied by the caller; this package does not parse it.
	Number string `json:"number"`
	// MediaType defaults to image when empty.
	MediaType string `json:"mediatype"`
	// MimeType defaults to image/jpeg when empty.
	MimeType string `json:"mimetype"`
	Caption  string `json:"caption"`
	// Media is the base64 payload expected by Evolution.
	Media    string `json:"media"`
	FileName string `json:"fileName,omitempty"`
}

// TextInput es el contrato exacto de Evolution para sendText.
type TextInput struct {
	Number string `json:"number"`
	// Text is sent verbatim to Evolution.
	Text string `json:"text"`
}

// Receipt normaliza la respuesta; Evolution varía entre versiones.
type Receipt struct {
	// Status is SENT unless Evolution returns another state.
	Status string `json:"status"`
	// MessageID is empty when Evolution omits key.id.
	MessageID string          `json:"messageId,omitempty"`
	Raw       json.RawMessage `json:"-"`
}

// APIError conserva el estado y un extracto del cuerpo para diagnóstico.
type APIError struct {
	StatusCode int
	Body       string
	Retryable  bool
	RetryAfter int // segundos; 0 si no vino
}

func (e *APIError) Error() string {
	return fmt.Sprintf("evolution api: estado %d: %s", e.StatusCode, e.Body)
}
