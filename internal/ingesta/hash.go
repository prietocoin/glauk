package ingesta

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
)

// hashFromMessage replica la identidad del binomio de remitHub: el SHA-256 del
// archivo que WhatsApp adjunta en fileSha256. Cuando la instancia no lo envía,
// se usa el id del mensaje para no perder el evento.
func hashFromMessage(body messageBody, key keyData) string {
	if body.ImageMessage != nil {
		if hash := decodeFileSha(body.ImageMessage.FileSha256); hash != "" {
			return hash
		}
	}
	id := strings.TrimSpace(key.ID)
	if id == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:])
}

// decodeFileSha entiende las tres formas conocidas: objeto índice→byte,
// arreglo de bytes y cadena en base64 o hexadecimal.
func decodeFileSha(raw json.RawMessage) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return ""
	}
	switch trimmed[0] {
	case '{':
		return decodeFileShaObject(trimmed)
	case '[':
		var values []int
		if err := json.Unmarshal(trimmed, &values); err != nil {
			return ""
		}
		return bytesToHex(values)
	case '"':
		var encoded string
		if err := json.Unmarshal(trimmed, &encoded); err != nil {
			return ""
		}
		return decodeFileShaString(encoded)
	}
	return ""
}

func decodeFileShaObject(raw []byte) string {
	var indexed map[string]int
	if err := json.Unmarshal(raw, &indexed); err == nil {
		values := make([]int, 0, 32)
		for index := 0; index < 32; index++ {
			if value, ok := indexed[strconv.Itoa(index)]; ok {
				values = append(values, value)
			}
		}
		if len(values) == 32 {
			return bytesToHex(values)
		}
	}

	var buffer struct {
		Data []int `json:"data"`
	}
	if err := json.Unmarshal(raw, &buffer); err == nil && len(buffer.Data) > 0 {
		return bytesToHex(buffer.Data)
	}
	return ""
}

func bytesToHex(values []int) string {
	if len(values) == 0 {
		return ""
	}
	buffer := make([]byte, len(values))
	for index, value := range values {
		buffer[index] = byte(value)
	}
	return hex.EncodeToString(buffer)
}

func decodeFileShaString(value string) string {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return ""
	}
	if len(clean) == 64 {
		if _, err := hex.DecodeString(clean); err == nil {
			return strings.ToLower(clean)
		}
	}
	encodings := []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding,
		base64.URLEncoding, base64.RawURLEncoding,
	}
	for _, encoding := range encodings {
		if decoded, err := encoding.DecodeString(clean); err == nil && len(decoded) > 0 {
			return hex.EncodeToString(decoded)
		}
	}
	return ""
}

func shortHash(hash string) string {
	clean := strings.ToUpper(strings.TrimSpace(hash))
	if len(clean) <= 8 {
		return clean
	}
	return clean[:8]
}
