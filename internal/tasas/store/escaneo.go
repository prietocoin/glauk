package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ErrNotFound es el centinela de "la consulta fue válida pero no había el registro".
var ErrNotFound = errors.New("registro no encontrado")

// queryMany ejecuta una consulta y traduce cada fila con mapRow; es el único camino
// de lectura del paquete y quien envuelve los fallos con %w, para que la causa del
// motor no se pierda.
func queryMany[T any](ctx context.Context, q Querier, query string, args []any, mapRow func(pgx.Rows) (T, error)) ([]T, error) {
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ejecutando la consulta: %w", err)
	}
	defer rows.Close()

	values := make([]T, 0, 2)
	for rows.Next() {
		value, err := mapRow(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorriendo las filas: %w", err)
	}
	return values, nil
}

// queryOne ejecuta una consulta que devuelve a lo sumo una fila, interpreta "no había
// filas" como ErrNotFound y toma la primera, igual que el servicio original con rows[0].
func queryOne[T any](ctx context.Context, q Querier, query string, args []any, mapRow func(pgx.Rows) (T, error)) (T, error) {
	values, err := queryMany(ctx, q, query, args, mapRow)
	var zero T
	if err != nil {
		return zero, err
	}
	if len(values) == 0 {
		return zero, ErrNotFound
	}
	return values[0], nil
}

// text convierte una columna de texto anulable en cadena: el esquema admite nulos en
// campos que el negocio trata como opcionales (id_grupo, moneda_base, rol).
func text(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
