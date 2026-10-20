// Package domain define el vocabulario compartido del negocio de tasas: el punto
// de acuerdo entre el calculador, la capa de datos y los handlers, para que ninguno
// de ellos tenga que conocer al resto.
package domain

// Porcentajes son los márgenes que un socio aplica sobre la tasa triangulada; su
// signo se ignora: el depósito siempre suma y el pago siempre resta.
type Porcentajes struct {
	Deposito float64 `json:"deposito"`
	Pago     float64 `json:"pago"`
}

// Moneda describe cómo publica un socio una divisa concreta.
// Activo es un puntero a propósito: el origen omite el campo y una divisa sin él
// debe seguir publicándose, mientras que false la excluye. Un bool simple no puede
// representar esa diferencia, y los fixtures dorados la fijan.
type Moneda struct {
	Activo     *bool       `json:"activo,omitempty"`
	Porcentaje Porcentajes `json:"porcentaje"`
}

// Perfil es un socio: su identidad, el grupo donde publica y sus divisas.
type Perfil struct {
	Nombre     string             `json:"nombre"`
	IDGrupo    string             `json:"id_grupo,omitempty"`
	MonedaBase string             `json:"moneda_base,omitempty"`
	Monedas    map[string]*Moneda `json:"monedas"`
}

// Lote es un conjunto de tasas publicado en un momento dado.
//
// Las tasas se guardan sin tipo fijo porque el origen mezcla números y cadenas con
// separador de miles ("4,050.50"); normalizar esa mezcla es tarea del calculador.
type Lote struct {
	IDTasa string         `json:"id_tasa,omitempty"`
	Tasas  map[string]any `json:"tasas"`
}

// TarjetaPais es una fila de la cartelera: una divisa con sus precios, su bandera
// y su tendencia frente al lote anterior.
type TarjetaPais struct {
	Code       string  `json:"code"`
	NombrePais string  `json:"nombre_pais"`
	Bandera    string  `json:"bandera"`
	Compra     float64 `json:"compra"`
	Venta      float64 `json:"venta"`
	Trend      string  `json:"trend"`
}

// Cartelera es el resultado del cálculo para un socio: lo que el renderizador
// convierte en imagen y lo que viaja como pie de foto.
type Cartelera struct {
	NombreSocio string        `json:"nombre_socio"`
	IDGrupo     string        `json:"id_grupo,omitempty"`
	MonedaBase  string        `json:"moneda_base"`
	LoteTasa    string        `json:"lote_tasa"`
	Fecha       string        `json:"fecha_actualizacion"`
	Hora        string        `json:"hora_actualizacion"`
	Tarjetas    []TarjetaPais `json:"tarjetas_paises"`
}
