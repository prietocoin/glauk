package evolution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"glauk/internal/platform/config"
)

const testAPIKey = "super-secret-key"

func newTestClient(t *testing.T, handler http.HandlerFunc, cfg config.EvolutionConfig) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	cfg.BaseURL = srv.URL
	cfg.APIKey = testAPIKey
	cfg.Instance = "Glauk"
	if cfg.Timeout == 0 {
		cfg.Timeout = 2 * time.Second
	}
	return New(cfg)
}

func TestSendTextSuccess(t *testing.T) {
	const body = `{"key":{"id":"WAMID-1"},"status":"PENDING"}`
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/message/sendText/Glauk" {
			t.Errorf("request inesperado: %s %s", r.Method, r.URL.Path)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if r.Header.Get("apikey") != testAPIKey {
			t.Errorf("apikey = %q", r.Header.Get("apikey"))
		}
		if r.Header.Get("Authorization") != "" || r.Header.Get("Global-Api-Key") != "" {
			t.Errorf("cabeceras de auth prohibidas presentes")
		}
		var in TextInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Number != "123" || in.Text != "hola" {
			t.Errorf("payload inesperado: %+v, err=%v", in, err)
		}
		_, _ = w.Write([]byte(body))
	}, config.EvolutionConfig{})

	receipt, err := client.SendText(context.Background(), TextInput{Number: "123", Text: "hola"})
	if err != nil {
		t.Fatalf("SendText: %v", err)
	}
	if receipt.MessageID != "WAMID-1" || receipt.Status != "PENDING" {
		t.Fatalf("recibo = %+v", receipt)
	}
	if string(receipt.Raw) != body {
		t.Fatalf("raw = %q", receipt.Raw)
	}
}

func TestSendTextEmptyOrUnstructuredBody(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"vacío", ""},
		{"sin envelope", "not-json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			}, config.EvolutionConfig{})
			receipt, err := client.SendText(context.Background(), TextInput{Number: "123", Text: "hola"})
			if err != nil {
				t.Fatalf("SendText: %v", err)
			}
			if receipt.Status != "SENT" || receipt.MessageID != "" {
				t.Fatalf("recibo = %+v", receipt)
			}
			if string(receipt.Raw) != tc.body {
				t.Fatalf("raw = %q", receipt.Raw)
			}
		})
	}
}

func TestNonRetryableStatusesTruncateBody(t *testing.T) {
	body := strings.Repeat("x", 600)
	for _, status := range []int{400, 401, 403, 404, 422} {
		t.Run(fmt.Sprintf("%d", status), func(t *testing.T) {
			var calls atomic.Int32
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.WriteHeader(status)
				_, _ = w.Write([]byte(body))
			}, config.EvolutionConfig{MaxRetries: 3, RetryBase: time.Millisecond})

			_, err := client.SendText(context.Background(), TextInput{Number: "123", Text: "hola"})
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error = %v", err)
			}
			if apiErr.StatusCode != status || apiErr.Retryable {
				t.Fatalf("apiErr = %+v", apiErr)
			}
			if len(apiErr.Body) != 500 || !strings.HasSuffix(apiErr.Body, "...") {
				t.Fatalf("cuerpo no truncado: len=%d", len(apiErr.Body))
			}
			if calls.Load() != 1 {
				t.Fatalf("llamadas = %d", calls.Load())
			}
		})
	}
}

func TestAPIKeyNeverAppearsInError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprintf(w, "clave rechazada: %s", r.Header.Get("apikey"))
	}, config.EvolutionConfig{})

	_, err := client.SendText(context.Background(), TextInput{Number: "123", Text: "hola"})
	if err == nil {
		t.Fatal("se esperaba error")
	}
	if strings.Contains(err.Error(), testAPIKey) {
		t.Fatalf("el error filtró la apikey: %v", err)
	}
}

func TestContextTimeoutAbortsImmediately(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}, config.EvolutionConfig{MaxRetries: 3, RetryBase: time.Millisecond})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := client.SendText(ctx, TextInput{Number: "123", Text: "hola"})
	elapsed := time.Since(start)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("el timeout no abortó a tiempo: %v", elapsed)
	}
}

func TestSendMediaDefaultsAndPayload(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/message/sendMedia/Glauk" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var in MediaInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Errorf("decode: %v", err)
		}
		if in.MediaType != "image" || in.MimeType != "image/jpeg" {
			t.Errorf("defaults = %q/%q", in.MediaType, in.MimeType)
		}
		_, _ = w.Write([]byte(`{"key":{"id":"MEDIA-1"}}`))
	}, config.EvolutionConfig{})

	receipt, err := client.SendMedia(context.Background(), MediaInput{
		Number: "123", Caption: "hola", Media: "base64",
	})
	if err != nil {
		t.Fatalf("SendMedia: %v", err)
	}
	if receipt.MessageID != "MEDIA-1" {
		t.Fatalf("recibo = %+v", receipt)
	}
}
