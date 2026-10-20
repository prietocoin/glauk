package model

// Perfil es una fila de perfiles_glaukov tal como la consume el panel.
//
// SaldoInicial se conserva como string para no perder la precisión que entregaba
// node-pg. Activo es un campo derivado de H-02: no tiene columna propia en el
// contrato y nunca se acepta desde una petición.
//
// El JSON usa las mismas claves que el contrato histórico (snake_case), porque
// el HTML de Perfiles lee socio.mostrar, socio.monedas y socio.id_grupo sin una
// capa de traducción. Monedas siempre se serializa como objeto, no como string.
// El orden de los campos no es contractual; la forma sí lo es. Un consumidor
// nuevo debe ignorar campos desconocidos para tolerar futuras extensiones.
type Perfil struct {
	IDGrupo      string  `json:"id_grupo"`
	Nombre       string  `json:"nombre"`
	Activo       bool    `json:"activo"`
	Rol          string  `json:"rol"`
	MonedaBase   string  `json:"moneda_base"`
	SaldoInicial string  `json:"saldo_inicial"`
	Mostrar      Mostrar `json:"mostrar"`
	Monedas      Monedas `json:"monedas"`
	Herencia     bool    `json:"herencia"`
}

// GuardarRequest es el payload de POST /api/v1/perfiles/guardar.
//
// Activo no existe aquí a propósito: si el cliente lo envía, encoding/json lo
// ignora y la actividad se recalcula con mostrar.tasas. Los alias históricos se
// resuelven en requests.go sin relajar esta regla.
type GuardarRequest struct {
	Nombre       string        `json:"nombre"`
	IDGrupo      string        `json:"id_grupo"`
	Rol          string        `json:"rol"`
	MonedaBase   string        `json:"moneda_base"`
	SaldoInicial FlexibleFloat `json:"saldo_inicial"`
	Mostrar      *Mostrar      `json:"mostrar"`
	Monedas      Monedas       `json:"monedas"`
	Herencia     bool          `json:"herencia"`
}
