package store

import (
	"context"
	"fmt"
	"strings"
)

// asignacion es una columna que el panel puede escribir y el valor que le toca.
// `coalesce` distingue el interruptor, donde un nulo significa "no lo toques".
type asignacion struct {
	columna  string
	valor    any
	coalesce bool
}

// sentenciaDeGuardado arma el UPDATE con las columnas que la tabla tiene de verdad.
//
// ¿Por qué no una sentencia literal? Porque el esquema heredado no es el que
// suponíamos: `activo` no existe en perfiles_glaukov y nombrarla hace fallar el
// guardado con 42703, el mismo error que tumbó la lectura. Preguntándole a la tabla
// qué columnas tiene, el guardado escribe las que existen y calla las que no.
func sentenciaDeGuardado(ctx context.Context, q Querier, cambios SocioUpdate, nombre string) (string, []any, error) {
	monedas, err := encodeJSON(cambios.Monedas)
	if err != nil {
		return "", nil, err
	}
	existentes, err := columnasDe(ctx, q, tablaSocios)
	if err != nil {
		return "", nil, err
	}

	candidatas := []asignacion{
		{columna: "monedas", valor: monedas},
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
		return "", nil, fmt.Errorf("la tabla %s no tiene ninguna columna que el panel escriba", tablaSocios)
	}
	args = append(args, nombre)
	sentencia := fmt.Sprintf("UPDATE %s SET %s WHERE LOWER(nombre) = LOWER($%d) RETURNING *", tablaSocios, strings.Join(partes, ", "), len(args))
	return sentencia, args, nil
}
