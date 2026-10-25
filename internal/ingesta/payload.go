package ingesta

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	// Key y Message se guardan crudos: Evolution necesita mediaKey, directPath,
	// url, fileEncSha256 y demás campos que un struct tipado descartaría.
	Key              json.RawMessage `json:"key"`
	Message          json.RawMessage `json:"message"`
	PushName         string          `json:"pushName"`
	MessageTimestamp flexInt64       `json:"messageTimestamp"`
	MessageType      string          `json:"messageType"`
	Instance         string          `json:"instance"`
	InstanceID       string          `json:"instanceId"`
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

	var key keyData
	if len(message.Key) > 0 {
		if err := json.Unmarshal(message.Key, &key); err != nil {
			return Event{}, fmt.Errorf("%w: key inválida: %v", ErrInvalidPayload, err)
		}
	}
	var body messageBody
	if len(message.Message) > 0 {
		if err := json.Unmarshal(message.Message, &body); err != nil {
			return Event{}, fmt.Errorf("%w: message inválido: %v", ErrInvalidPayload, err)
		}
	}

	if body.ImageMessage == nil && message.MessageType != "imageMessage" {
		return Event{}, ErrNotImage
	}
	if key.FromMe {
		return Event{}, ErrNotImage
	}

	hash := hashFromMessage(body, key)
	if hash == "" {
		return Event{}, fmt.Errorf("%w: no se pudo calcular el hash", ErrInvalidPayload)
	}
	caption := ""
	if body.ImageMessage != nil {
		caption = body.ImageMessage.Caption
	}
	return Event{
		Instancia:  normalizeInstance(firstNonEmpty(instance, message.Instance, message.InstanceID, defaultInstance)),
		HashLargo:  normalizeHash(hash),
		HashCorto:  shortHash(hash),
		Usuario:    firstNonEmpty(key.ParticipantAlt, key.Participant, key.RemoteJID),
		Grupo:      key.RemoteJID,
		NombrePush: strings.TrimSpace(message.PushName),
		Caption:    cleanCaption(caption),
		Timestamp:  int64(message.MessageTimestamp),
		Key:        message.Key,
		Message:    message.Message,
		MimeType:   mimeOf(body.ImageMessage),
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
