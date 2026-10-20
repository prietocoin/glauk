package store

import (
	"context"
	"fmt"
)

// tablaSocios es la tabla heredada de perfiles. Vive en un solo lugar porque la
// nombran la lectura, el guardado y la introspección del esquema.
const tablaSocios = "perfiles_glaukov"

// columnasDe pregunta a la tabla qué columnas tiene de verdad.
//
// Se pide `SELECT * ... LIMIT 0`: no devuelve filas, solo la descripción del
// resultado, así que se descubre exactamente el conjunto que verá un `SELECT *` real,
// respetando el search_path y sin depender de information_schema ni de permisos
// distintos. Cuesta una consulta por guardado y a cambio el guardado deja de inventar
// columnas: `activo` no existe en esta base, y nombrarla en un UPDATE falla con 42703
// igual que fallaba la lectura.
func columnasDe(ctx context.Context, q Querier, tabla string) (map[string]bool, error) {
	rows, err := q.Query(ctx, fmt.Sprintf("SELECT * FROM %s LIMIT 0", tabla))
	if err != nil {
		return nil, fmt.Errorf("consultando las columnas de %s: %w", tabla, err)
	}
	defer rows.Close()

	campos := rows.FieldDescriptions()
	columnas := make(map[string]bool, len(campos))
	for _, campo := range campos {
		columnas[campo.Name] = true
	}
	return columnas, nil
}
