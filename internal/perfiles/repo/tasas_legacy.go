package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ActualizacionSocioTasas es el contrato mínimo del adaptador legado de Tasas.
//
// No es parte del modelo de Perfiles: agrupa las columnas que el panel histórico
// de Tasas podía cambiar. El adaptador vive aquí porque perfiles_glaukov tiene un
// único escritor; Tasas conserva su API, pero no vuelve a construir SQL de escritura.
type ActualizacionSocioTasas struct {
	Monedas    []byte
	Activo     *bool
	ModoPrueba *bool
	Rol        *string
}

// SentenciaSocioTasas construye el UPDATE con las columnas que existen de verdad.
//
// `activo` y `modo_prueba` son opcionales en el esquema heredado: el llamador pasa
// el conjunto de columnas descubierto con introspección. Los interruptores usan
// COALESCE porque un puntero nil significa "no lo toques"; `monedas` siempre se
// escribe y `rol` también es opcional. La tabla se fija aquí para que ningún otro
// paquete pueda redirigir esta escritura a otra relación.
func SentenciaSocioTasas(existentes map[string]bool, cambios ActualizacionSocioTasas, nombre string) (string, []any, error) {
	candidatas := []struct {
		columna  string
		valor    any
		coalesce bool
	}{
		{columna: "monedas", valor: cambios.Monedas},
		{columna: "activo", valor: cambios.Activo, coalesce: true},
		{columna: "modo_prueba", valor: cambios.ModoPrueba, coalesce: true},
		{columna: "rol", valor: cambios.Rol, coalesce: true},
	}
	partes := make([]string, 0, len(candidatas))
	args := make([]any, 0, len(candidatas)+1)
	for _, candidata := range candidatas {
		if !existentes[candidata.columna] {
			continue
		}
		args = append(args, candidata.valor)
		if candidata.coalesce {
			partes = append(partes, fmt.Sprintf("%s = COALESCE($%d, %s)", candidata.columna, len(args), candidata.columna))
			continue
		}
		partes = append(partes, fmt.Sprintf("%s = $%d", candidata.columna, len(args)))
	}
	if len(partes) == 0 {
		return "", nil, fmt.Errorf("la tabla perfiles_glaukov no tiene ninguna columna que el panel escriba")
	}
	args = append(args, nombre)
	sentencia := fmt.Sprintf(
		"UPDATE perfiles_glaukov SET %s WHERE LOWER(nombre) = LOWER($%d) RETURNING *",
		strings.Join(partes, ", "), len(args),
	)
	return sentencia, args, nil
}

// ActualizarSocioTasas ejecuta el UPDATE en el único paquete autorizado a escribir
// perfiles_glaukov y devuelve las filas para que Tasas las escanee con su contrato.
// El llamador debe cerrar el pgx.Rows devuelto.
func ActualizarSocioTasas(ctx context.Context, q Querier, existentes map[string]bool, cambios ActualizacionSocioTasas, nombre string) (pgx.Rows, error) {
	sentencia, args, err := SentenciaSocioTasas(existentes, cambios, nombre)
	if err != nil {
		return nil, err
	}
	return q.Query(ctx, sentencia, args...)
}
