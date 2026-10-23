package ingesta

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrNotImage permite que el webhook responda 200 e ignore textos y audios.
var ErrNotImage = errors.New("ingesta: el mensaje no es una imagen")

// ErrInvalidPayload separa un 400 del cliente de un fallo interno (500).
var ErrInvalidPayload = errors.New("ingesta: payload inválido")

type envelope struct {
	Instance string          `json:"instance"`
	Data     json.RawMessage `json:"data"`
	Body     json.RawMessage `json:"body"`
}

type messageData struct {
	Key              keyData     `json:"key"`
	Message          messageBody `json:"message"`
	PushName         string      `json:"pushName"`
	MessageTimestamp flexInt64   `json:"messageTimestamp"`
	MessageType      string      `json:"messageType"`
	Instance         string      `json:"instance"`
	InstanceID       string      `json:"instanceId"`
}

type keyData struct {
	RemoteJID      string `json:"remoteJid"`
	FromMe         bool   `json:"fromMe"`
	ID             string `json:"id"`
	Participant    string `json:"participant"`
	ParticipantAlt string `json:"participantAlt"`
}

type messageBody struct {
	ImageMessage *imageMessage `json:"imageMessage"`
}

type imageMessage struct {
	Caption    string          `json:"caption"`
	Mimetype   string          `json:"mimetype"`
	FileSha256 json.RawMessage `json:"fileSha256"`
}

// ParseEvent acepta las variantes v1 (body), v2 (data) y los arreglos que
// entrega Evolution. El hash se deriva de fileSha256 o, si falta, del id.
func ParseEvent(raw []byte, defaultInstance string) (Event, error) {
	data, instance, err := locateData(raw, 0)
	if err != nil {
		return Event{}, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	var message messageData
	if err := json.Unmarshal(data, &message); err != nil {
		return Event{}, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	if message.Message.ImageMessage == nil && message.MessageType != "imageMessage" {
		return Event{}, ErrNotImage
	}
	if message.Key.FromMe {
		return Event{}, ErrNotImage
	}

	hash := hashFromMessage(message)
	if hash == "" {
		return Event{}, fmt.Errorf("%w: no se pudo calcular el hash", ErrInvalidPayload)
	}
	caption := ""
	if message.Message.ImageMessage != nil {
		caption = message.Message.ImageMessage.Caption
	}
	return Event{
		Instancia:  normalizeInstance(firstNonEmpty(instance, message.Instance, message.InstanceID, defaultInstance)),
		HashLargo:  hash,
		HashCorto:  shortHash(hash),
		Usuario:    firstNonEmpty(message.Key.ParticipantAlt, message.Key.Participant, message.Key.RemoteJID),
		Grupo:      message.Key.RemoteJID,
		NombrePush: strings.TrimSpace(message.PushName),
		Caption:    cleanCaption(caption),
		Timestamp:  int64(message.MessageTimestamp),
		Key:        mustRaw(message.Key),
		Message:    mustRaw(message.Message),
		MimeType:   mimeOf(message.Message.ImageMessage),
	}, nil
}

func locateData(raw []byte, depth int) (json.RawMessage, string, error) {
	if depth > 4 {
		return nil, "", errors.New("ingesta: payload demasiado anidado")
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, "", errors.New("ingesta: payload vacío")
	}
	if trimmed[0] == '[' {
		var items []json.RawMessage
		if err := json.Unmarshal(trimmed, &items); err != nil || len(items) == 0 {
			return nil, "", errors.New("ingesta: arreglo de eventos vacío")
		}
		return locateData(items[0], depth+1)
	}

	var env envelope
	if err := json.Unmarshal(trimmed, &env); err != nil {
		return nil, "", err
	}
	if len(env.Data) > 0 {
		return firstObject(env.Data), env.Instance, nil
	}
	if len(env.Body) > 0 {
		if data, instance, err := locateData(env.Body, depth+1); err == nil {
			return data, firstNonEmpty(env.Instance, instance), nil
		}
		return firstObject(env.Body), env.Instance, nil
	}
	return trimmed, env.Instance, nil
}

func firstObject(raw json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var items []json.RawMessage
		if err := json.Unmarshal(trimmed, &items); err == nil && len(items) > 0 {
			return items[0]
		}
	}
	return trimmed
}

// flexInt64 acepta segundos, milisegundos y cadenas numéricas.
type flexInt64 int64

func (f *flexInt64) UnmarshalJSON(data []byte) error {
	value := strings.Trim(strings.TrimSpace(string(data)), `"`)
	if value == "" || value == "null" {
		*f = 0
		return nil
	}
	number, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		float, floatErr := strconv.ParseFloat(value, 64)
		if floatErr != nil {
			return fmt.Errorf("timestamp inválido: %q", value)
		}
		number = int64(float)
	}
	switch {
	case number > 1_000_000_000_000_000:
		number /= 1_000_000
	case number > 1_000_000_000_000:
		number /= 1_000
	}
	*f = flexInt64(number)
	return nil
}

func mustRaw(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage("{}")
	}
	return raw
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeInstance(value string) string {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return "glauk"
	}
	return clean
}

func cleanCaption(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func mimeOf(image *imageMessage) string {
	if image == nil {
		return "image/jpeg"
	}
	return firstNonEmpty(image.Mimetype, "image/jpeg")
}
