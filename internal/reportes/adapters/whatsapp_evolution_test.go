package adapters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"glauk/internal/platform/config"
	"glauk/internal/platform/evolution"
	"glauk/internal/reportes"
)

func TestWhatsAppEvolutionSimulado(t *testing.T) {
	client := evolution.New(config.EvolutionConfig{})
	adapter := NewWhatsAppEvolution(client)
	image, err := adapter.EnviarImagen(context.Background(), reportes.MensajeImagen{
		Destino: "120363012345678901@g.us", Base64: "abc", MimeType: "image/jpeg",
		Caption: "reporte", FileName: "reporte.jpg",
	})
	if err != nil || image.MessageID == "" || image.Estado != "SIMULATED" || !image.Simulado {
		t.Fatalf("imagen = %+v err=%v", image, err)
	}
	text, err := adapter.EnviarTexto(context.Background(), reportes.MensajeTexto{
		Destino: "120363012345678901@g.us", Texto: "hola",
	})
	if err != nil || text.MessageID == "" || text.Estado != "SIMULATED" || !text.Simulado {
		t.Fatalf("texto = %+v err=%v", text, err)
	}
}

func TestWhatsAppEvolutionErrores(t *testing.T) {
	var nilAdapter *WhatsAppEvolution
	if _, err := nilAdapter.EnviarImagen(context.Background(), reportes.MensajeImagen{}); err == nil {
		t.Fatal("adaptador nil debe fallar")
	}
	if _, err := NewWhatsAppEvolution(nil).EnviarTexto(context.Background(), reportes.MensajeTexto{}); err == nil {
		t.Fatal("cliente nil debe fallar")
	}
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "sin conexión", http.StatusInternalServerError)
	}))
	defer servidor.Close()
	client := evolution.New(config.EvolutionConfig{
		BaseURL: servidor.URL, Timeout: time.Second, MaxRetries: 0,
	})
	adapter := NewWhatsAppEvolution(client)
	if _, err := adapter.EnviarImagen(context.Background(), reportes.MensajeImagen{
		Destino: "1@g.us", Base64: "abc",
	}); err == nil {
		t.Fatal("un 500 de Evolution debe propagarse")
	}
	if _, err := adapter.EnviarTexto(context.Background(), reportes.MensajeTexto{
		Destino: "1@g.us", Texto: "hola",
	}); err == nil {
		t.Fatal("un 500 de texto debe propagarse")
	}
}

func TestWiringEnsamblaModulo(t *testing.T) {
	module := New(fakeSocioReader{}, &fakeReportReader{}, nil, reportes.Config{})
	if !module.Valido() || module.Idempotency == nil || module.RateLimiter == nil {
		t.Fatalf("módulo mal ensamblado: %+v", module)
	}
}
