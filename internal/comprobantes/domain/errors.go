// Package domain contiene las entidades y reglas estables de Comprobantes.
//
// No importa Fiber, pgx ni paquetes de infraestructura: las capas externas
// traducen sus errores y formatos a estos contratos.
package domain

import (
	"errors"
	"fmt"
)

var (
	// ErrComprobanteNotFound indica que el hash solicitado no existe.
	ErrComprobanteNotFound = errors.New("comprobante no encontrado")
	// ErrQueueItemNotFound indica que la cola no contiene el hash solicitado.
	ErrQueueItemNotFound = errors.New("registro de cola no encontrado")
)

// ValidationError representa un payload o parámetro inválido del dominio.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// NewValidation construye un error de validación con texto público.
func NewValidation(message string) error {
	return &ValidationError{Message: message}
}

// Validationf construye un error de validación formateado.
func Validationf(format string, args ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}
