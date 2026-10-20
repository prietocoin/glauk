package store

import "encoding/json"

// textoDe devuelve el texto de una columna, y "" cuando no vino o vino nula.
//
// Acepta las formas en que el valor puede llegar: cadena, puntero del driver y bytes
// crudos. La ausencia no es un error: `id_grupo`, `moneda_base` y `rol` son opcionales
// en el original (`socio.id_grupo || JID_GRUPO_PRUEBA`, `s.rol || 'SOCIO'`).
func textoDe(fila map[string]any, columna string) string {
	switch valor := fila[columna].(type) {
	case string:
		return valor
	case *string:
		return text(valor)
	case []byte:
		return string(valor)
	default:
		return ""
	}
}

// booleanoDe devuelve el interruptor de una columna, o nil si la columna no vino.
//
// nil significa "la tabla no tiene ese dato", que no es lo mismo que falso: el
// original solo apagaba a un socio cuando el valor era exactamente `false`
// (`socio.activo === false`), y esa distinción se conserva aquí.
func booleanoDe(fila map[string]any, columna string) *bool {
	switch valor := fila[columna].(type) {
	case bool:
		return &valor
	case *bool:
		return valor
	default:
		return nil
	}
}

// jsonDe devuelve los bytes de una columna JSON, sea jsonb o text.
//
// El driver entrega la columna ya decodificada (un objeto de Go) cuando el tipo de
// destino es `any`; aquí se vuelve a serializar para que la interpretación quede en
// una sola función, decodeJSON, que sirve igual para jsonb y para text. El original
// se defendía de ambos con `typeof s.monedas === 'string'`.
func jsonDe(fila map[string]any, columna string) ([]byte, error) {
	switch valor := fila[columna].(type) {
	case nil:
		return nil, nil
	case []byte:
		return valor, nil
	case string:
		return []byte(valor), nil
	default:
		return json.Marshal(valor)
	}
}
