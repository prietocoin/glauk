package model

// Canonica completa una Moneda construida en Go, fuera de UnmarshalJSON.
//
// El codec de escritura no puede volver a pasar por JSON: al hacerlo, el
// porcentaje cero que MarshalJSON siempre emite parecería un porcentaje
// explícito y borraría un depósito/pago programático. Esta función distingue la
// forma de Tasas (porcentaje presente) de la forma plana de Perfiles y sincroniza
// ambos lados sin perder la prioridad H-01: si el JSON trajo porcentaje, ese
// valor manda incluso cuando es cero; si la Moneda se construyó en Go, los
// campos planos siguen siendo la entrada.
func (m Moneda) Canonica() Moneda {
	if m.porcentajePresente || m.Porcentaje.Deposito != 0 || m.Porcentaje.Pago != 0 {
		m.Deposito, m.Pago = m.Porcentaje.Deposito, m.Porcentaje.Pago
	} else {
		m.Porcentaje = Porcentajes{Deposito: m.Deposito, Pago: m.Pago}
	}
	if m.Def == "" {
		m.Def = primero(m.Tipo, "D")
	}
	if m.Tipo == "" {
		m.Tipo = m.Def
	}
	depOp := primero(m.DepOp, m.PolarD, m.Polaridad, "+")
	pagoOp := primero(m.PagoOp, m.PolarP, contraparte(depOp))
	m.DepOp = depOp
	m.PagoOp = pagoOp
	m.PolarD = primero(m.PolarD, depOp, "+")
	m.PolarP = primero(m.PolarP, pagoOp, "-")
	m.Polaridad = primero(m.PolarD, m.Polaridad, depOp, "+")
	return m
}
