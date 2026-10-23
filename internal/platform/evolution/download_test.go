package evolution

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"glauk/internal/platform/config"
)

func TestDownloadMediaPreservaMediaKey(t *testing.T) {
	key := json.RawMessage(`{"id":"ABC123","remoteJid":"1@g.us","fromMe":false}`)
	message := json.RawMessage(`{"imageMessage":{"mediaKey":"media-key-critica","directPath":"/v/t62/abc","url":"https://mmg.whatsapp.net/abc"}}`)

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("método = %s", r.Method)
		}
		if r.URL.Path != "/message/getBase64FromMediaMessage/Glauk" {
			t.Errorf("ruta = %s", r.URL.Path)
		}
		var body struct {
			Message struct {
				Key     json.RawMessage `json:"key"`
				Message json.RawMessage `json:"message"`
			} `json:"message"`
			ConvertToMp4 bool `json:"convertToMp4"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("payload: %v", err)
		}
		if string(body.Message.Key) != string(key) {
			t.Errorf("key alterada: %s", body.Message.Key)
		}
		var media struct {
			ImageMessage struct {
				MediaKey string `json:"mediaKey"`
			} `json:"imageMessage"`
		}
		if err := json.Unmarshal(body.Message.Message, &media); err != nil {
			t.Fatalf("message: %v", err)
		}
		if media.ImageMessage.MediaKey != "media-key-critica" {
			t.Errorf("mediaKey perdida: %q", media.ImageMessage.MediaKey)
		}
		if body.ConvertToMp4 {
			t.Error("convertToMp4 debe ser false")
		}
		_, _ = w.Write([]byte(`{"base64":"aGVsbG8=","mimetype":"image/jpeg"}`))
	}, config.EvolutionConfig{})

	data, mime, err := client.DownloadMedia(context.Background(), "Glauk", key, message)
	if err != nil {
		t.Fatalf("DownloadMedia: %v", err)
	}
	if string(data) != "hello" || mime != "image/jpeg" {
		t.Fatalf("data=%q mime=%q", data, mime)
	}
}
