package hoo

import (
	"encoding/json"
	"fmt"
)

// Mapear traduce la respuesta cruda del origen al borrador que consume el panel.
//
// Reproduce el mapeo del controlador legado, incluidas sus rarezas: cada valor puede
// llegar como número o como texto, y lo ausente cae al valor por defecto. Es una
// función pura —entran bytes, sale el borrador—, así que se prueba sin red.
func Mapear(datos []byte) (Cambios, error) {
	item, err := primero(datos)
	if err != nil {
		return Cambios{}, err
	}
	return Cambios{
		ID:        texto(item, "id"),
		ARS:       texto(item, "ars"),
		BOB:       numero(item, "bob", 0),
		BRL:       numero(item, "brl", 0),
		CAD:       numero(item, "cad", 0),
		CLP:       texto(item, "clp"),
		COP:       texto(item, "cop"),
		CRC:       numero(item, "crc", 0),
		DOP:       numero(item, "dop", 0),
		EUR:       numero(item, "eur", 0),
		MXN:       numero(item, "mxn", 0),
		PEN:       numero(item, "pen", 0),
		PYG:       numero(item, "pyg", 0),
		VES:       texto(item, "ves"),
		USD:       numero(item, "usd", 1),
		USDT:      numero(item, "usdt", 1),
		PYUSD:     numero(item, "pyusd", 1.2),
		ECU:       numero(item, "ecu", 1),
		PAN:       numero(item, "pan", 1),
		CREATEDAT: fecha(item),
	}, nil
}

// primero devuelve el registro del cuerpo: el origen manda un objeto con las tasas o
// un array del que solo interesa el primer elemento.
func primero(datos []byte) (map[string]any, error) {
	var decodificado any
	if err := json.Unmarshal(datos, &decodificado); err != nil {
		return nil, fmt.Errorf("respuesta inesperada de HOO: %w", err)
	}
	switch valor := decodificado.(type) {
	case map[string]any:
		return valor, nil
	case []any:
		if len(valor) > 0 {
			if item, esObjeto := valor[0].(map[string]any); esObjeto {
				return item, nil
			}
		}
	}
	return map[string]any{}, nil
}
