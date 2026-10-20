package hoo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// timeout es el plazo propio del cliente, independiente del que imponga quien llama:
// el panel no debe quedarse esperando a un origen externo que no responde.
const timeout = 30 * time.Second

// Cliente consulta el origen global de tasas por HTTP y devuelve el borrador mapeado.
type Cliente struct {
	url  string
	http *http.Client
}

// Cliente cumple el contrato que el handler espera, y esta afirmación lo obliga a
// seguir cumpliéndolo aunque cambie quien lo inyecte.
var _ Borrador = (*Cliente)(nil)

// New construye el cliente para una URL de origen.
func New(url string) *Cliente {
	return &Cliente{url: url, http: &http.Client{Timeout: timeout}}
}

// Obtener consulta el origen y devuelve el borrador ya traducido.
//
// El estado se valida antes de leer el cuerpo y el JSON se mapea aparte, en una
// función pura: si el origen cambia de forma, el fallo aparece en el mapeo y no
// escondido detrás de una petición de red.
func (c *Cliente) Obtener(ctx context.Context) (Cambios, error) {
	peticion, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return Cambios{}, fmt.Errorf("construir la petición a HOO: %w", err)
	}
	respuesta, err := c.http.Do(peticion)
	if err != nil {
		return Cambios{}, fmt.Errorf("consultar HOO: %w", err)
	}
	defer respuesta.Body.Close()

	if respuesta.StatusCode != http.StatusOK {
		return Cambios{}, fmt.Errorf("HOO respondió estado %d", respuesta.StatusCode)
	}
	datos, err := io.ReadAll(respuesta.Body)
	if err != nil {
		return Cambios{}, fmt.Errorf("leer la respuesta de HOO: %w", err)
	}
	return Mapear(datos)
}
