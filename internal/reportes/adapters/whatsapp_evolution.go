package adapters

import (
	"context"
	"errors"

	"glauk/internal/platform/evolution"
	"glauk/internal/reportes"
)

// WhatsAppEvolution adapta el cliente HTTP de Evolution al puerto de envío.
type WhatsAppEvolution struct {
	client *evolution.Client
}

// NewWhatsAppEvolution construye el adaptador con el cliente ya cableado.
func NewWhatsAppEvolution(client *evolution.Client) *WhatsAppEvolution {
	return &WhatsAppEvolution{client: client}
}

// EnviarImagen despacha un JPEG validado manteniendo el recibo del dominio.
func (a *WhatsAppEvolution) EnviarImagen(ctx context.Context,
	msg reportes.MensajeImagen) (reportes.Recibo, error) {
	if a == nil || a.client == nil {
		return reportes.Recibo{}, errors.New("cliente de Evolution no configurado")
	}
	receipt, err := a.client.SendMedia(ctx, evolution.MediaInput{
		Number: msg.Destino, MediaType: "image", MimeType: msg.MimeType,
		Caption: msg.Caption, Media: msg.Base64, FileName: msg.FileName,
	})
	if err != nil {
		return reportes.Recibo{}, err
	}
	return reportes.Recibo{
		MessageID: receipt.MessageID, Estado: receipt.Status,
		Simulado: a.client.Simulated(),
	}, nil
}

// EnviarTexto despacha un reporte textual por el mismo cliente.
func (a *WhatsAppEvolution) EnviarTexto(ctx context.Context,
	msg reportes.MensajeTexto) (reportes.Recibo, error) {
	if a == nil || a.client == nil {
		return reportes.Recibo{}, errors.New("cliente de Evolution no configurado")
	}
	receipt, err := a.client.SendText(ctx, evolution.TextInput{
		Number: msg.Destino, Text: msg.Texto,
	})
	if err != nil {
		return reportes.Recibo{}, err
	}
	return reportes.Recibo{
		MessageID: receipt.MessageID, Estado: receipt.Status,
		Simulado: a.client.Simulated(),
	}, nil
}
