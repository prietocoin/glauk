package reportes

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Service orquesta los casos de uso de reportes. No conoce HTTP, SQL ni el
// cliente de Evolution: solo habla con los tres puertos de ports.go.
type Service struct {
	socios      SociosProvider
	movimientos MovimientosProvider
	sender      WhatsAppSender
	cfg         Config
	now         func() time.Time
}

// NewService construye el dominio y normaliza la configuración ausente.
func NewService(socios SociosProvider, movimientos MovimientosProvider,
	sender WhatsAppSender, cfg Config) *Service {
	return &Service{
		socios: socios, movimientos: movimientos, sender: sender,
		cfg: cfg.withDefaults(), now: time.Now,
	}
}

// Filtros combina ambas fuentes. Si el adaptador de movimientos falla, el error
// se propaga: un modal con filtros incompletos produce reportes incorrectos.
func (s *Service) Filtros(ctx context.Context, rol string) (Filtros, error) {
	return s.movimientos.Filtros(ctx, strings.TrimSpace(rol))
}

// EnviarImagen valida el JPEG y resuelve el destino antes de tocar WhatsApp.
// El caption se recorta a 1024 caracteres y el nombre se sanitiza a 120 runas.
func (s *Service) EnviarImagen(ctx context.Context, in SolicitudImagen) (Resultado, error) {
	destino, err := s.resolverDestino(ctx, in.Socio, in.RemoteJID)
	if err != nil {
		return Resultado{}, err
	}

	img, err := DecodificarImagen(in.Base64, s.cfg)
	if err != nil {
		return Resultado{}, err
	}

	caption := strings.TrimSpace(in.Caption)
	if len(caption) > 1024 {
		caption = caption[:1024]
	}
	nombre := SanitizarNombre(in.FileName)

	recibo, err := s.sender.EnviarImagen(ctx, MensajeImagen{
		Destino: destino, Base64: img.Base64, MimeType: "image/jpeg",
		Caption: caption, FileName: nombre,
	})
	if err != nil {
		return Resultado{}, errors.Join(ErrEnvioFallido, err)
	}

	return Resultado{
		JID: destino, MessageID: recibo.MessageID, Estado: recibo.Estado,
		Simulado: recibo.Simulado, Bytes: len(img.Bytes),
	}, nil
}

// EnviarTexto recorta el texto a 4096 caracteres y resuelve el destino.
// Un texto vacío se rechaza antes de consultar el directorio o el sender.
func (s *Service) EnviarTexto(ctx context.Context, in SolicitudTexto) (Resultado, error) {
	texto := strings.TrimSpace(in.Texto)
	if texto == "" {
		return Resultado{}, ErrTextoRequerido
	}
	if len(texto) > 4096 {
		texto = texto[:4096]
	}

	destino, err := s.resolverDestino(ctx, in.Socio, in.RemoteJID)
	if err != nil {
		return Resultado{}, err
	}

	recibo, err := s.sender.EnviarTexto(ctx, MensajeTexto{Destino: destino, Texto: texto})
	if err != nil {
		return Resultado{}, errors.Join(ErrEnvioFallido, err)
	}

	return Resultado{
		JID: destino, MessageID: recibo.MessageID, Estado: recibo.Estado,
		Simulado: recibo.Simulado,
	}, nil
}
