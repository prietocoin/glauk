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
	m.DepOp = primero(m.DepOp, m.PolarD, "+")
	m.PagoOp = primero(m.PagoOp, m.PolarP, "-")
	m.PolarD = primero(m.PolarD, m.DepOp, "+")
	m.PolarP = primero(m.PolarP, m.PagoOp, "-")
	m.Polaridad = primero(m.Polaridad, m.PolarD, "+")
	return m
}
