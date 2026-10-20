// Package reportes implementa el dominio de entrega de reportes.
//
// Este paquete NO conoce PostgreSQL, perfiles, comprobantes ni tasas. Toda
// dependencia externa se declara aquí como puerto y se satisface en
// internal/reportes/adapters, que se cablea en cmd/server.
package reportes

import (
	"context"
	"time"
)

// Socio es la proyección mínima del directorio que el dominio necesita.
// No es la fila de perfiles_glaukov: es lo que reportes puede entender.
type Socio struct {
	Nombre  string
	Rol     string
	IDGrupo string // JID de WhatsApp; puede venir vacío o inválido
	Activo  bool
	Moneda  string
}

// Filtros alimenta el modal del panel. Los roles y entidades vienen del
// directorio; los hashes, del historial de movimientos.
type Filtros struct {
	Roles     []string
	Entidades []string
	Hashes    []string
}

// ConsultaMovimientos es el filtro normalizado que el adaptador traduce a su
// propio lenguaje. Las fechas son inclusivas.
type ConsultaMovimientos struct {
	Rol     string
	Entidad string
	Hash    string
	Desde   time.Time
	Hasta   time.Time
	Orden   string // "fecha_asc" | "fecha_desc"
	Limite  int
}

// Movimiento es una fila de auditoría ya proyectada. No incluye secretos ni
// columnas internas.
type Movimiento struct {
	HashLargo  string
	HashCorto  string
	Fecha      time.Time
	Socio      string
	Rol        string
	Moneda     string
	Monto      float64
	TipoOp     string
	Banco      string
	Referencia string
	Estado     string
}

// Resumen es el agregado que puede ir en un reporte de texto.
type Resumen struct {
	Total      int
	MontoTotal float64
	PorMoneda  map[string]float64
	Desde      time.Time
	Hasta      time.Time
}

// SociosProvider resuelve el directorio sin exponer su persistencia.
type SociosProvider interface {
	SocioPorNombre(ctx context.Context, nombre string) (Socio, error)
	Socios(ctx context.Context, rol string) ([]Socio, error)
}

// MovimientosProvider entrega filtros y movimientos ya proyectados.
type MovimientosProvider interface {
	Filtros(ctx context.Context, rol string) (Filtros, error)
	Movimientos(ctx context.Context, q ConsultaMovimientos) ([]Movimiento, error)
	Resumen(ctx context.Context, q ConsultaMovimientos) (Resumen, error)
}

// MensajeImagen es el contrato de salida hacia WhatsApp. Base64 va sin el
// prefijo data: y sin espacios; el adaptador lo traduce al JSON de Evolution.
type MensajeImagen struct {
	Destino  string
	Base64   string
	MimeType string
	Caption  string
	FileName string
}

// MensajeTexto es el contrato de salida para reportes sin imagen.
type MensajeTexto struct {
	Destino string
	Texto   string
}

// Recibo normaliza la respuesta del proveedor.
type Recibo struct {
	MessageID string
	Estado    string
	Simulado  bool
}

// WhatsAppSender es el único puerto de salida hacia mensajería.
type WhatsAppSender interface {
	EnviarImagen(ctx context.Context, msg MensajeImagen) (Recibo, error)
	EnviarTexto(ctx context.Context, msg MensajeTexto) (Recibo, error)
}
