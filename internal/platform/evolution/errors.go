package evolution

import (
	"encoding/json"
	"strconv"
	"strings"
)

func classify(status int, raw []byte, retryAfter string) *APIError {
	body := snippet(raw, 500)
	retry := status == 408 || status == 425 || status == 429 || status >= 500
	seconds := 0
	if retryAfter != "" {
		if parsed, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && parsed > 0 {
			seconds = parsed
		}
	}
	return &APIError{StatusCode: status, Body: body, Retryable: retry, RetryAfter: seconds}
}

// snippet acota el cuerpo para que un error no vuelque una respuesta completa.
func snippet(raw []byte, max int) string {
	text := strings.TrimSpace(string(raw))
	if max <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

// parseReceipt acepta respuestas vacías o de versiones distintas sin fallar.
func parseReceipt(raw []byte) Receipt {
	receipt := Receipt{Status: "SENT", Raw: append([]byte(nil), raw...)}
	var envelope struct {
		Key struct {
			ID string `json:"id"`
		} `json:"key"`
		MessageID string `json:"messageId"`
		ID        string `json:"id"`
		Status    string `json:"status"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return receipt
	}
	switch {
	case envelope.Key.ID != "":
		receipt.MessageID = envelope.Key.ID
	case envelope.MessageID != "":
		receipt.MessageID = envelope.MessageID
	case envelope.ID != "":
		receipt.MessageID = envelope.ID
	}
	if envelope.Status != "" {
		receipt.Status = envelope.Status
	}
	return receipt
}
