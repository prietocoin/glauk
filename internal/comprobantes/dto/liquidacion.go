package dto

import "glauk/internal/comprobantes/domain"

// Liquidacion es el DTO congelado que devuelven liquidar y actualizar.
//
// Es un alias del tipo de dominio para que la respuesta HTTP conserve los
// nombres legacy sin mantener una segunda definición de campos. El panel lee
// siempre monto_1/tasa_1/me1 y monto_2/tasa_2/me2, incluso cuando el segundo
// socio es GENERAL y llega con importes en cero.
//
// Ejemplo:
//
//	{"hash_largo":"abc","socio_1":"FUNDDA","tipo_op1":"D-COP",
//	 "monto_1":100,"tasa_1":4000,"me1":100,"socio_2":null,
//	 "tipo_op2":"D-COP","monto_2":0,"tasa_2":1,"me2":0,"lote_tasa":"T052"}
type Liquidacion = domain.Liquidacion

// LiquidacionPtr facilita respuestas que necesitan distinguir ausencia.
// El puntero nunca se serializa como cero: nil sigue siendo null.
//
// El DTO no copia ni normaliza valores: si el dominio congeló un monto con
// signo negativo, la respuesta HTTP conserva ese signo exacto para que el
// panel no lo recalcule con una regla distinta a la del backend.
//
// Tampoco se agregan campos derivados como totales o etiquetas: cualquier
// cálculo adicional debe vivir en calc y probarse con los mismos fixtures
// dorados del snapshot. La forma del JSON es parte del contrato público y
// cualquier cambio exige actualizar el panel y los tests de rutas.
type LiquidacionPtr = *domain.Liquidacion
