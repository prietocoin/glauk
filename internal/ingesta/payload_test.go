package ingesta

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
)

func hashBytes() []byte {
	values := make([]byte, 32)
	for index := range values {
		values[index] = byte(index)
	}
	return values
}

func imagePayload(sha any, instance string) []byte {
	data := map[string]any{
		"key": map[string]any{
			"remoteJid":   "1203630@g.us",
			"participant": "584140000000@s.whatsapp.net",
			"id":          "ABC123",
			"fromMe":      false,
		},
		"message": map[string]any{
			"imageMessage": map[string]any{
				"caption":       "  pago   recibido ",
				"mimetype":      "image/png",
				"fileSha256":    sha,
				"mediaKey":      "media-key-critica",
				"directPath":    "/v/t62.7118-24/abc",
				"url":           "https://mmg.whatsapp.net/v/t62.7118-24/abc",
				"fileEncSha256": []int{9, 8, 7},
			},
		},
		"pushName":         "Ana",
		"messageTimestamp": 1700000000000,
		"messageType":      "imageMessage",
	}
	raw, _ := json.Marshal(map[string]any{"instance": instance, "data": data})
	return raw
}

func indexedSha() map[string]int {
	values := map[string]int{}
	for index, value := range hashBytes() {
		values[strconv.Itoa(index)] = int(value)
	}
	return values
}

func TestParseEventV2(t *testing.T) {
	event, err := ParseEvent(imagePayload(indexedSha(), "Glauk"), "glauk")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	wantHash := hex.EncodeToString(hashBytes())
	if event.HashLargo != wantHash {
		t.Fatalf("hash = %s, want %s", event.HashLargo, wantHash)
	}
	if event.HashCorto != "00010203" {
		t.Fatalf("hash corto = %s", event.HashCorto)
	}
	if event.Instancia != "Glauk" {
		t.Fatalf("instancia = %s", event.Instancia)
	}
	if event.Usuario != "584140000000@s.whatsapp.net" {
		t.Fatalf("usuario = %s", event.Usuario)
	}
	if event.Grupo != "1203630@g.us" {
		t.Fatalf("grupo = %s", event.Grupo)
	}
	if event.Caption != "pago recibido" {
		t.Fatalf("caption = %q", event.Caption)
	}
	if event.Timestamp != 1_700_000_000 {
		t.Fatalf("timestamp = %d", event.Timestamp)
	}
	if event.MimeType != "image/png" {
		t.Fatalf("mime = %s", event.MimeType)
	}
}

func TestParseEventPreservaMediaCruda(t *testing.T) {
	event, err := ParseEvent(imagePayload(indexedSha(), "glauk"), "glauk")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var message struct {
		ImageMessage struct {
			MediaKey   string `json:"mediaKey"`
			DirectPath string `json:"directPath"`
			URL        string `json:"url"`
		} `json:"imageMessage"`
	}
	if err := json.Unmarshal(event.Message, &message); err != nil {
		t.Fatalf("message crudo: %v", err)
	}
	if message.ImageMessage.MediaKey != "media-key-critica" {
		t.Fatalf("mediaKey perdida: %q", message.ImageMessage.MediaKey)
	}
	if message.ImageMessage.DirectPath == "" || message.ImageMessage.URL == "" {
		t.Fatalf("message recortado: %+v", message.ImageMessage)
	}
	var key map[string]any
	if err := json.Unmarshal(event.Key, &key); err != nil {
		t.Fatalf("key cruda: %v", err)
	}
	if key["id"] != "ABC123" {
		t.Fatalf("key recortada: %+v", key)
	}
}

func TestParseEventAcceptsArrayAndBody(t *testing.T) {
	data := imagePayload([]int{1, 2, 3}, "jairo")
	wrapped, _ := json.Marshal(map[string]any{"body": json.RawMessage(data)})
	array, _ := json.Marshal([]json.RawMessage{wrapped})

	event, err := ParseEvent(array, "glauk")
	if err != nil {
		t.Fatalf("parse array: %v", err)
	}
	if event.Instancia != "jairo" {
		t.Fatalf("instancia = %s", event.Instancia)
	}
	if event.HashLargo != hex.EncodeToString([]byte{1, 2, 3}) {
		t.Fatalf("hash = %s", event.HashLargo)
	}
}

func TestParseEventAcceptsBase64Sha(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString(hashBytes())
	event, err := ParseEvent(imagePayload(encoded, "glauk"), "glauk")
	if err != nil {
		t.Fatalf("parse base64: %v", err)
	}
	if event.HashLargo != hex.EncodeToString(hashBytes()) {
		t.Fatalf("hash = %s", event.HashLargo)
	}
}

func TestParseEventFallsBackToKeyID(t *testing.T) {
	event, err := ParseEvent(imagePayload(nil, "glauk"), "glauk")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	sum := sha256.Sum256([]byte("ABC123"))
	if event.HashLargo != hex.EncodeToString(sum[:]) {
		t.Fatalf("hash de respaldo = %s", event.HashLargo)
	}
}

func TestParseEventRejectsNonImage(t *testing.T) {
	raw := []byte(`{"instance":"glauk","data":{"key":{"remoteJid":"1@s"},"message":{"conversation":"hola"},"messageType":"conversation"}}`)
	if _, err := ParseEvent(raw, "glauk"); !errors.Is(err, ErrNotImage) {
		t.Fatalf("err = %v, want ErrNotImage", err)
	}
}

func TestParseEventRejectsInvalidJSON(t *testing.T) {
	if _, err := ParseEvent([]byte(`{`), "glauk"); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}
