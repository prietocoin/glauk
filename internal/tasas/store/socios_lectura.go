package store

import "context"

// Las lecturas de socios piden la fila entera, como el servicio original, y no una
// lista de columnas elegida por nosotros.
//
// Escribir aquí las columnas que creemos que existen fue justo lo que rompió /socios
// en producción: la tabla heredada no tiene `activo`, y una lista explícita convierte
// cualquier desacuerdo del esquema en un 42703 que tumba el endpoint. Con `*` la
// consulta se adapta a la tabla real y el mapeo por nombre decide, columna a columna,
// qué llegó. Leyendo por nombre, la misma función sirve para la tabla que hay y para
// la que venga: el contrato es el nombre de la columna, no su posición. Una columna
// ausente se traduce al valor cero, que es lo que hacía el original al leer
// `socio.activo` de un objeto al que le faltaba la propiedad.
const (
	sqlSocioByName = `SELECT * FROM ` + tablaSocios + ` WHERE LOWER(nombre) = LOWER($1)`
	sqlAllSocios   = `SELECT * FROM ` + tablaSocios + ` ORDER BY nombre ASC`
)

// SocioByName busca un socio por nombre sin distinguir mayúsculas.
//
// La comparación se resuelve en la base, con LOWER, y no trayendo la tabla a memoria.
func SocioByName(ctx context.Context, q Querier, nombre string) (Socio, error) {
	return queryOne(ctx, q, sqlSocioByName, []any{nombre}, scanSocio)
}

// AllSocios devuelve todos los socios ordenados por nombre, que es el orden en que
// el panel los muestra y el que evita que la lista cambie de orden entre recargas.
func AllSocios(ctx context.Context, q Querier) ([]Socio, error) {
	return queryMany(ctx, q, sqlAllSocios, nil, scanSocio)
}
