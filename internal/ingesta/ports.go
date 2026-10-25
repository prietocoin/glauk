package ingesta

import (
	"context"
	"encoding/json"

	"glauk/internal/platform/gemini"
)

// MediaFetcher descarga el binario desde Evolution. La implementación real es
// platform/evolution.Client; las pruebas pueden devolver bytes en memoria.
type MediaFetcher interface {
	DownloadMedia(ctx context.Context, instance string, key,
		message json.RawMessage) ([]byte, string, error)
}

// ImageStore abstrae Cloudflare R2. PublicURL existe para no obligar al
// pipeline a construir URLs a mano cuando el dominio público no está definido.
type ImageStore interface {
	Put(ctx context.Context, key string, data []byte, contentType string) (string, error)
	PublicURL(key string) string
}

// Extractor es el puerto de Gemini.
type Extractor interface {
	Extract(ctx context.Context, prompt string, image []byte, mimeType string) (gemini.Result, error)
}

// Liquidador calcula el snapshot con el motor interno de tasas y perfiles.
type Liquidador interface {
	Liquidar(ctx context.Context, in LiquidacionInput) (LiquidacionResult, error)
}

// Store es la puerta a PostgreSQL que usa el servicio. Se declara como
// interfaz para poder sustituirla en pruebas sin pgxmock en cada caso.
type Store interface {
	Registrar(ctx context.Context, event Event) (Registro, error)
	Claim(ctx context.Context, hash string) (bool, error)
	ImagenExistente(ctx context.Context, hash string) (string, error)
	ActualizarImagen(ctx context.Context, hash, url string) error
	Contexto(ctx context.Context, hash string) (Contexto, error)
	GuardarExtraccion(ctx context.Context, hash, instancia, url string,
		extraccion gemini.Result) error
	GuardarLiquidacion(ctx context.Context, result LiquidacionResult) error
	MarcarEstado(ctx context.Context, hash string, estado Estado) error
}
