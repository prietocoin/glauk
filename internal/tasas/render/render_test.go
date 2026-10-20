package render

import (
	"bytes"
	"image/png"
	"math"
	"testing"

	"glauk/internal/tasas/domain"
)

// carteleraDePrueba arma una cartelera con la cantidad de países pedida y deja la
// primera fila con una tasa no finita: es el caso que más rompe el dibujo, porque
// el cálculo la entrega como NaN y la vista la convierte en nil.
func carteleraDePrueba(cantidad int) domain.Cartelera {
	tarjetas := make([]domain.TarjetaPais, 0, cantidad)
	for indice := 0; indice < cantidad; indice++ {
		tarjetas = append(tarjetas, domain.TarjetaPais{
			Code:       "ARS",
			NombrePais: "Argentina",
			Bandera:    "ar",
			Compra:     1250,
			Venta:      1310,
			Trend:      "up",
		})
	}
	if cantidad > 0 {
		tarjetas[0].Compra = math.NaN()
	}
	return domain.Cartelera{
		NombreSocio: "Glauk",
		LoteTasa:    "T077",
		Fecha:       "19/10/2026",
		Hora:        "10:30",
		Tarjetas:    tarjetas,
	}
}

// TestGenerarCarteleraDevuelvePNGValido comprueba el contrato del endpoint: lo que
// sale del dibujo se puede decodificar como PNG y mide lo que promete la fórmula.
func TestGenerarCarteleraDevuelvePNGValido(t *testing.T) {
	salida, err := GenerarCartelera(carteleraDePrueba(12))
	if err != nil {
		t.Fatalf("GenerarCartelera: %v", err)
	}
	imagen, err := png.Decode(bytes.NewReader(salida))
	if err != nil {
		t.Fatalf("el resultado no es un PNG válido: %v", err)
	}
	if ancho := imagen.Bounds().Dx(); ancho != anchoLienzo {
		t.Errorf("ancho = %d, se esperaba %d", ancho, anchoLienzo)
	}
	if alto := imagen.Bounds().Dy(); alto != 1280 {
		t.Errorf("alto = %d, se esperaba 1280", alto)
	}
}

// TestGenerarCarteleraSinPaises evita la regresión de un socio que no publica
// divisas: el lienzo mínimo debe salir igual, sin filas ni banderas.
func TestGenerarCarteleraSinPaises(t *testing.T) {
	salida, err := GenerarCartelera(carteleraDePrueba(0))
	if err != nil {
		t.Fatalf("GenerarCartelera: %v", err)
	}
	imagen, err := png.Decode(bytes.NewReader(salida))
	if err != nil {
		t.Fatalf("el resultado no es un PNG válido: %v", err)
	}
	if alto := imagen.Bounds().Dy(); alto != int(altoMinimo) {
		t.Errorf("alto = %d, se esperaba %d", alto, int(altoMinimo))
	}
}

// TestAltoDelLienzoRespetaLasFilas fija la fórmula heredada: una base de 260 px,
// una fila de 170 por cada dos tarjetas y un mínimo de 450.
func TestAltoDelLienzoRespetaLasFilas(t *testing.T) {
	casos := []struct {
		paises int
		alto   int
	}{
		{0, 450},
		{1, 450},
		{2, 450},
		{4, 600},
		{6, 770},
		{12, 1280},
		{13, 1450},
	}
	for _, caso := range casos {
		if obtenido := altoDelLienzo(caso.paises); obtenido != caso.alto {
			t.Errorf("%d países: alto = %d, se esperaba %d", caso.paises, obtenido, caso.alto)
		}
	}
}
