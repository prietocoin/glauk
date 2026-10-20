// Package hoo define el contrato con el origen externo de tasas (HOO).
//
// El panel pide un "borrador" —los valores de cambio del día— que el usuario revisa
// y corrige antes de guardar un lote. Aquí vive solo el contrato y la forma de la
// respuesta; el cliente HTTP y el mapeo de los veinte campos llegan en la F6, de
// modo que el handler pueda existir y probarse antes que su origen.
package hoo

import "context"

// Borrador entrega los valores de cambio con los que se arma un lote nuevo.
//
// Se declara como interfaz, y no como un tipo concreto, para que el handler de
// /hoo-borrador pueda probarse con un doble en memoria: la prueba de rutas no debe
// depender de que un servicio externo esté en pie.
type Borrador interface {
	// Obtener consulta el origen y devuelve los valores ya mapeados. Respeta el
	// contexto: si el panel cierra la pestaña, la consulta se abandona.
	Obtener(ctx context.Context) (Cambios, error)
}

// Cambios es el borrador de un lote nuevo, con los veinte campos que el panel
// muestra antes de guardar.
//
// Cada campo conserva el tipo con el que el origen lo publicaba: unos viajaban como
// texto (ID, ARS, CLP, COP, VES) y otros como número. Es un detalle del mapeo y no
// una decisión de diseño, y por eso queda escrito aquí: "arreglarlo" obligaría a que
// el panel convirtiera valores que hoy recibe listos.
//
// Los numéricos son punteros porque el origen puede entregar un texto que no sea un
// número: allí el resultado era NaN y JSON.stringify lo publicaba como null. Un
// float64 no puede representar eso sin romper la serialización, así que el nulo se
// declara en el tipo en lugar de descubrirse al responder.
type Cambios struct {
	ID        string   `json:"ID"`
	ARS       string   `json:"ARS"`
	BOB       *float64 `json:"BOB"`
	BRL       *float64 `json:"BRL"`
	CAD       *float64 `json:"CAD"`
	CLP       string   `json:"CLP"`
	COP       string   `json:"COP"`
	CRC       *float64 `json:"CRC"`
	DOP       *float64 `json:"DOP"`
	EUR       *float64 `json:"EUR"`
	MXN       *float64 `json:"MXN"`
	PEN       *float64 `json:"PEN"`
	PYG       *float64 `json:"PYG"`
	VES       string   `json:"VES"`
	USD       *float64 `json:"USD"`
	USDT      *float64 `json:"USDT"`
	PYUSD     *float64 `json:"PYUSD"`
	ECU       *float64 `json:"ECU"`
	PAN       *float64 `json:"PAN"`
	CREATEDAT string   `json:"CREATED_AT"`
}
