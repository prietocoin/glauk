package ingesta

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"glauk/internal/platform/config"
	"glauk/internal/platform/gemini"
)

const objectPrefix = "comprobantes/"

// Service orquesta el pipeline. No contiene SQL ni fórmulas: delega en Store,
// en los clientes externos y en el Liquidador.
type Service struct {
	store    Store
	media    MediaFetcher
	images   ImageStore
	ai       Extractor
	liq      Liquidador
	cfg      config.IngestaConfig
	instance string
	log      *slog.Logger
}

// NewService construye el caso de uso con todas sus dependencias explícitas.
func NewService(store Store, media MediaFetcher, images ImageStore,
	ai Extractor, liq Liquidador, cfg config.IngestaConfig,
	instance string, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		store: store, media: media, images: images, ai: ai, liq: liq,
		cfg: cfg, instance: normalizeInstance(instance), log: log,
	}
}

// Recibir valida el binomio en PostgreSQL y devuelve el Job que el Module
// encolará. La respuesta HTTP nunca espera a Gemini ni a R2.
func (s *Service) Recibir(ctx context.Context, raw []byte) (Decision, *Job, error) {
	event, err := ParseEvent(raw, s.instance)
	if errors.Is(err, ErrNotImage) {
		return Decision{Status: "ignored"}, nil, nil
	}
	if err != nil {
		return Decision{}, nil, err
	}
	if ignoredInstance(event.Instancia) {
		return Decision{Status: "ignored", Hash: event.HashLargo}, nil, nil
	}

	registro, err := s.store.Registrar(ctx, event)
	if err != nil {
		return Decision{}, nil, err
	}
	if registro.Conteo < 2 {
		return Decision{
			Status: "waiting_pair", Hash: event.HashLargo, Conteo: registro.Conteo,
		}, nil, nil
	}
	if !registro.Completo && registro.Estado != EstadoFallo {
		return Decision{Status: "duplicate", Hash: event.HashLargo, Conteo: registro.Conteo}, nil, nil
	}

	claimed, err := s.store.Claim(ctx, event.HashLargo)
	if err != nil {
		return Decision{}, nil, err
	}
	if !claimed {
		return Decision{Status: "duplicate", Hash: event.HashLargo, Conteo: registro.Conteo}, nil, nil
	}
	job := &Job{Event: event}
	return Decision{Status: "queued", Hash: event.HashLargo, Conteo: registro.Conteo}, job, nil
}

// Procesar ejecuta un trabajo completo. Es idempotente: si la extracción ya
// existe reutiliza esos datos y solo reintenta la liquidación pendiente.
func (s *Service) Procesar(ctx context.Context, job Job) error {
	contexto, err := s.store.Contexto(ctx, job.HashLargo)
	if err != nil {
		return err
	}
	instancia := firstNonEmpty(job.Instancia, contexto.Instancia, s.instance)
	url := contexto.URLImagen
	needExtraction := !contexto.Procesado || contexto.Monto <= 0

	var image []byte
	var mimeType string
	if needExtraction {
		image, mimeType, err = s.obtenerImagen(ctx, job)
		if err != nil {
			return err
		}
		if strings.TrimSpace(url) == "" || !strings.HasPrefix(url, "http") {
			key := objectPrefix + job.HashLargo + ".jpg"
			url, err = s.images.Put(ctx, key, image, mimeType)
			if err != nil {
				return fmt.Errorf("ingesta: subir a R2: %w", err)
			}
		}
		if err := s.store.ActualizarImagen(ctx, job.HashLargo, url); err != nil {
			return err
		}
	}

	extraccion := Extraccion{
		Monto: contexto.Monto, Moneda: contexto.Moneda, Banco: contexto.Banco,
		Referencia: contexto.Referencia, Titular: contexto.Titular,
	}
	if needExtraction {
		caption := firstNonEmpty(job.Caption, contexto.Caption)
		prompt := gemini.PromptWithCaption(gemini.DefaultPrompt, caption)
		extraccion, err = s.ai.Extract(ctx, prompt, image, mimeType)
		if err != nil {
			return fmt.Errorf("ingesta: extraer con Gemini: %w", err)
		}
		if err := s.store.GuardarExtraccion(ctx, job.HashLargo, instancia, url, extraccion); err != nil {
			return err
		}
	}

	if extraccion.Monto <= 0 {
		return errors.New("ingesta: el comprobante no tiene un monto legible")
	}
	input := LiquidacionInput{
		HashLargo: job.HashLargo, HashCorto: firstNonEmpty(job.HashCorto, contexto.HashCorto),
		Instancia: instancia, Monto: extraccion.Monto, Moneda: extraccion.Moneda,
		Banco: extraccion.Banco, Titular: extraccion.Titular,
		Referencia: extraccion.Referencia, URLImagen: url,
		Caption:   firstNonEmpty(job.Caption, contexto.Caption),
		Timestamp: firstNonZero(job.Timestamp, contexto.Timestamp),
		Grupos:    contexto.Grupos,
	}
	result, err := s.liq.Liquidar(ctx, input)
	if err != nil {
		return fmt.Errorf("ingesta: liquidar: %w", err)
	}
	if err := s.store.GuardarLiquidacion(ctx, input, result, s.cfg.EscribirFinal); err != nil {
		return err
	}
	return s.store.MarcarEstado(ctx, job.HashLargo, EstadoProcesado)
}

func (s *Service) obtenerImagen(ctx context.Context, job Job) ([]byte, string, error) {
	var lastErr error
	if len(job.Key) > 0 && len(job.Message) > 0 && s.media != nil {
		data, mimeType, err := s.media.DownloadMedia(ctx, job.Instancia, job.Key, job.Message)
		if err == nil && len(data) > 0 {
			return data, mimeType, nil
		}
		lastErr = err
	}
	if s.images != nil {
		key := objectPrefix + job.HashLargo + ".jpg"
		data, mimeType, err := s.images.Get(ctx, key)
		if err == nil && len(data) > 0 {
			return data, mimeType, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("sin origen de imagen")
	}
	return nil, "", fmt.Errorf("ingesta: obtener imagen: %w", lastErr)
}

// Fallar deja el registro en FALLO para que un reenvío del webhook lo
// vuelva a encolar. Usa un contexto propio porque el del job ya expiró.
func (s *Service) Fallar(job Job) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.store.MarcarEstado(ctx, job.HashLargo, EstadoFallo); err != nil {
		s.log.Warn("ingesta: no se pudo marcar FALLO", "hash", job.HashLargo, "error", err)
	}
}

func ignoredInstance(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "john", "jhon":
		return true
	}
	return false
}

func firstNonZero(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
