// Package ingesta implementa el pipeline nativo que absorbe a remitHub y a
// glaukapi: webhook de Evolution, validación del binomio en PostgreSQL, R2,
// Gemini y liquidación con el motor interno de tasas.
package ingesta

import (
	"encoding/json"

	"glauk/internal/comprobantes/calc"
	"glauk/internal/comprobantes/domain"
	"glauk/internal/platform/gemini"
)

// Estado es el ciclo de vida de un registro en registros_raw.
type Estado string

const (
	EstadoRecibido  Estado = "RECIBIDO"
	EstadoEnCola    Estado = "EN_COLA"
	EstadoProcesado Estado = "PROCESADO"
	EstadoFallo     Estado = "FALLO"
)

// Event es un mensaje de Evolution ya normalizado. Key y Message se conservan
// crudos porque son el contrato exacto que exige getBase64FromMediaMessage.
type Event struct {
	Instancia  string
	HashLargo  string
	HashCorto  string
	Usuario    string
	Grupo      string
	NombrePush string
	Caption    string
	Timestamp  int64
	Key        json.RawMessage
	Message    json.RawMessage
	MimeType   string
}

// Job viaja por el canal interno; no se serializa ni se persiste.
type Job struct {
	Event
	Intentos int
}

// Decision es la respuesta inmediata del webhook.
type Decision struct {
	Status string
	Hash   string
	Conteo int
}

// Registro refleja el estado acumulado de registros_raw. Completo indica que
// esta llamada cerró el binomio (impacto 2), no que el pipeline terminó.
type Registro struct {
	Conteo   int
	Estado   Estado
	Completo bool
}

// Contexto reúne lo que el worker necesita de PostgreSQL antes de llamar a
// los servicios externos.
type Contexto struct {
	HashLargo  string
	HashCorto  string
	Instancia  string
	URLImagen  string
	Caption    string
	Timestamp  int64
	Monto      float64
	Moneda     string
	Banco      string
	Titular    string
	Referencia string
	Procesado  bool
	Grupos     []string
}

// Extraccion es el contrato que devuelve Gemini y se escribe en crudo.
type Extraccion = gemini.Result

// LiquidacionInput es la entrada mínima del adaptador de tasas.
type LiquidacionInput struct {
	HashLargo  string
	HashCorto  string
	Instancia  string
	Monto      float64
	Moneda     string
	Banco      string
	Titular    string
	Referencia string
	URLImagen  string
	Caption    string
	Timestamp  int64
	Grupos     []string
}

// LiquidacionResult conserva el snapshot canónico y los datos de perfil que
// alimentan la tabla legacy comprobantes_final.
type LiquidacionResult struct {
	Snapshot    domain.Liquidacion
	TipoManual  string
	Rol1        string
	Rol2        string
	MonedaBase1 string
	MonedaBase2 string
	Polaridad1  string
	Polaridad2  string
	Perfil1     calc.Profile
	Perfil2     calc.Profile
	Grupo1      string
	Grupo2      string
}
