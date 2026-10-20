package store

import (
	"encoding/json"
	"fmt"
)

// decodeJSON interpreta una columna JSON en el tipo que se pida.
//
// Se escanea la columna como bytes crudos y se interpreta aquí, en lugar de dejar
// que el driver adivine el tipo de destino, por dos razones. La primera es que así
// funciona igual si la columna es jsonb o text: el servicio original tenía que
// defenderse de ambas con typeof x === 'string', y esa duda desaparece al leer los
// bytes. La segunda es que convierte el caso difícil de la persistencia en una
// función pura, verificable sin base de datos.
//
// Una columna vacía se traduce al valor cero del tipo pedido, sin error: el esquema
// puede no tener el dato, y eso no es un JSON inválido.
func decodeJSON[T any](raw []byte, column string) (T, error) {
	var value T
	if len(raw) == 0 {
		return value, nil
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return value, fmt.Errorf("la columna %s no contiene JSON válido: %w", column, err)
	}
	return value, nil
}

// encodeJSON serializa un valor para enviarlo a una columna JSON.
//
// No se usa serialización manual en ningún punto del paquete: un mapa de tasas mal
// armado a mano es un error silencioso que solo se descubre publicando una cartelera
// con precios equivocados.
func encodeJSON(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("serializando el valor para la base de datos: %w", err)
	}
	return raw, nil
}
