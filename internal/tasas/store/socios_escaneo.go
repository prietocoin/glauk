package store

import (
	"fmt"

	"github.com/jackc/pgx/v5"

	"glauk/internal/tasas/domain"
)

// filaDe arma el mapa columna → valor de la fila actual.
//
// El esquema heredado no es fijo: la misma consulta puede traer `activo` o no
// traerlo, así que se pregunta por nombre y lo ausente queda en su valor cero, como
// el `undefined` del servicio original.
func filaDe(rows pgx.Rows) (map[string]any, error) {
	valores, err := rows.Values()
	if err != nil {
		return nil, fmt.Errorf("leyendo la fila: %w", err)
	}
	campos := rows.FieldDescriptions()
	fila := make(map[string]any, len(campos))
	for i, campo := range campos {
		fila[campo.Name] = valores[i]
	}
	return fila, nil
}

// scanSocio traduce una fila de socio leyendo las columnas por nombre, incluidas las
// que la tabla no tenga: `activo`, `modo_prueba`, `rol` e `id_grupo` son opcionales en
// el original y aquí lo siguen siendo. El rol se devuelve tal cual; aplicarle SOCIO
// por defecto es presentación y vive en la capa que arma la respuesta.
func scanSocio(rows pgx.Rows) (Socio, error) {
	fila, err := filaDe(rows)
	if err != nil {
		return Socio{}, err
	}
	crudo, err := jsonDe(fila, "monedas")
	if err != nil {
		return Socio{}, fmt.Errorf("leyendo las monedas del socio: %w", err)
	}
	monedas, err := decodeJSON[map[string]*domain.Moneda](crudo, "monedas")
	if err != nil {
		return Socio{}, err
	}
	mostrarRaw, err := jsonDe(fila, "mostrar")
	if err != nil {
		return Socio{}, fmt.Errorf("leyendo la visibilidad del socio: %w", err)
	}
	mostrar, err := decodeJSON[Mostrar](mostrarRaw, "mostrar")
	if err != nil {
		return Socio{}, err
	}
	return Socio{
		Perfil: domain.Perfil{
			Nombre:     textoDe(fila, "nombre"),
			IDGrupo:    textoDe(fila, "id_grupo"),
			MonedaBase: textoDe(fila, "moneda_base"),
			Monedas:    monedas,
		},
		Activo:     booleanoDe(fila, "activo"),
		ModoPrueba: booleanoDe(fila, "modo_prueba"),
		Rol:        textoDe(fila, "rol"),
		Mostrar:    mostrar,
	}, nil
}
