package evolution

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"glauk/internal/platform/config"
)

type panicTransport struct{}

func (panicTransport) RoundTrip(*http.Request) (*http.Response, error) {
	panic("simulación intentó usar la red")
}

func newSimulatedClient() *Client {
	client := New(config.EvolutionConfig{APIKey: "must-not-be-used", Instance: "Glauk"})
	client.http.Transport = panicTransport{}
	return client
}

func TestSimulationTextIsDeterministicAndOffline(t *testing.T) {
	client := newSimulatedClient()
	if !client.Simulated() {
		t.Fatal("BaseURL vacía debía activar simulación")
	}

	first, err := client.SendText(context.Background(), TextInput{Number: "123", Text: "hola"})
	if err != nil {
		t.Fatalf("SendText: %v", err)
	}
	second, err := client.SendText(context.Background(), TextInput{Number: "123", Text: "hola"})
	if err != nil {
		t.Fatalf("SendText: %v", err)
	}
	other, err := client.SendText(context.Background(), TextInput{Number: "123", Text: "adiós"})
	if err != nil {
		t.Fatalf("SendText: %v", err)
	}

	if first.Status != "SIMULATED" || !strings.HasPrefix(first.MessageID, "sim-") || len(first.MessageID) != 20 {
		t.Fatalf("recibo = %+v", first)
	}
	if first.MessageID != second.MessageID {
		t.Fatalf("ids no deterministas: %q != %q", first.MessageID, second.MessageID)
	}
	if first.MessageID == other.MessageID {
		t.Fatal("contenidos distintos generaron el mismo id")
	}
}

func TestSimulationMediaAndValidation(t *testing.T) {
	client := newSimulatedClient()
	receipt, err := client.SendMedia(context.Background(), MediaInput{
		Number: "123", Media: "base64", Caption: "hola",
	})
	if err != nil {
		t.Fatalf("SendMedia: %v", err)
	}
	if receipt.Status != "SIMULATED" || !strings.HasPrefix(receipt.MessageID, "sim-") {
		t.Fatalf("recibo = %+v", receipt)
	}

	if _, err := client.SendText(context.Background(), TextInput{Number: "123"}); err == nil {
		t.Fatal("text vacío debía fallar")
	}
	if _, err := client.SendText(context.Background(), TextInput{Text: "hola"}); err == nil {
		t.Fatal("number vacío debía fallar")
	}
	if _, err := client.SendMedia(context.Background(), MediaInput{Number: "123"}); err == nil {
		t.Fatal("media vacía debía fallar")
	}
	if _, err := client.SendMedia(context.Background(), MediaInput{Media: "base64"}); err == nil {
		t.Fatal("number vacío debía fallar")
	}
}
