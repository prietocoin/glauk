package reportes

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func jpegBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatalf("codificar JPEG: %v", err)
	}
	return buf.Bytes()
}

func jpegBase64(t *testing.T, width, height int) string {
	t.Helper()
	return base64.StdEncoding.EncodeToString(jpegBytes(t, width, height))
}

func TestDecodificarImagenValida(t *testing.T) {
	raw := jpegBase64(t, 240, 320)
	img, err := DecodificarImagen(raw, Config{})
	if err != nil {
		t.Fatalf("DecodificarImagen: %v", err)
	}
	if img.Ancho != 240 || img.Alto != 320 {
		t.Fatalf("dimensiones = %dx%d, quiero 240x320", img.Ancho, img.Alto)
	}
	if img.Base64 != base64.StdEncoding.EncodeToString(img.Bytes) {
		t.Fatal("base64 no quedó normalizado")
	}
	if len(img.Bytes) == 0 {
		t.Fatal("bytes vacíos")
	}
}

func TestDecodificarImagenDataURL(t *testing.T) {
	raw := "data:image/jpeg;base64," + jpegBase64(t, 240, 240)
	if _, err := DecodificarImagen(raw, Config{}); err != nil {
		t.Fatalf("data URL JPEG: %v", err)
	}
	if _, err := DecodificarImagen("DATA:IMAGE/JPEG;BASE64,"+jpegBase64(t, 240, 240), Config{}); err != nil {
		t.Fatalf("data URL mayúsculas: %v", err)
	}
}

func TestLimpiarDataURL(t *testing.T) {
	if got, err := LimpiarDataURL("  abc  "); err != nil || got != "abc" {
		t.Fatalf("base64 puro = %q, %v", got, err)
	}
	if _, err := LimpiarDataURL("data:image/png;base64,abc"); !errors.Is(err, ErrImagenFormato) {
		t.Fatalf("PNG data URL: %v", err)
	}
	if _, err := LimpiarDataURL("data:image/jpeg,abc"); !errors.Is(err, ErrImagenFormato) {
		t.Fatalf("data URL sin base64: %v", err)
	}
	if _, err := LimpiarDataURL("data:image/jpeg;base64"); !errors.Is(err, ErrImagenFormato) {
		t.Fatalf("data URL sin coma: %v", err)
	}
}

func TestDecodificarImagenRechazaFormatos(t *testing.T) {
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, image.NewRGBA(image.Rect(0, 0, 240, 240))); err != nil {
		t.Fatalf("codificar PNG: %v", err)
	}
	cases := []string{
		base64.StdEncoding.EncodeToString(pngBuf.Bytes()),
		base64.StdEncoding.EncodeToString([]byte("RIFF....WEBPVP8 ")),
		"hello",
		"data:image/svg+xml;base64,PHN2Zy8+",
	}
	for _, raw := range cases {
		if _, err := DecodificarImagen(raw, Config{}); !errors.Is(err, ErrImagenFormato) {
			t.Errorf("formato %q: err = %v, quiero ErrImagenFormato", raw, err)
		}
	}
}

func TestDecodificarImagenRechazaTamanoYDimensiones(t *testing.T) {
	raw := jpegBase64(t, 240, 240)
	cases := []struct {
		name string
		cfg  Config
		raw  string
	}{
		{"bytes", Config{ImageMaxBytes: 100}, raw},
		{"min", Config{}, jpegBase64(t, 100, 100)},
		{"max", Config{}, jpegBase64(t, 4200, 240)},
		{"pixeles", Config{ImageMaxPixels: 1000}, raw},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := DecodificarImagen(tc.raw, tc.cfg); !errors.Is(err, ErrImagenDimensiones) && !errors.Is(err, ErrImagenMuyGrande) {
				t.Fatalf("err = %v, quiero dimensiones o tamaño", err)
			}
		})
	}
}

func TestDecodificarImagenSinBase64(t *testing.T) {
	for _, raw := range []string{"", "   ", "data:image/jpeg;base64,"} {
		if _, err := DecodificarImagen(raw, Config{}); !errors.Is(err, ErrImagenRequerida) {
			t.Errorf("raw %q: err = %v", raw, err)
		}
	}
}

func TestSanitizarNombre(t *testing.T) {
	cases := map[string]string{
		`..\..\reporte.jpg`: "reporte.jpg",
		"a\x00b\n.jpg":      "ab.jpg",
		"  nombre.png  ":    "nombre.png",
	}
	for raw, want := range cases {
		if got := SanitizarNombre(raw); got != want {
			t.Errorf("SanitizarNombre(%q) = %q, quiero %q", raw, got, want)
		}
	}
	if got := SanitizarNombre(strings.Repeat("ñ", 130)); len([]rune(got)) != 120 {
		t.Fatalf("largo = %d runas, quiero 120", len([]rune(got)))
	}
}

func TestSanitizarNombreVacio(t *testing.T) {
	if got := SanitizarNombre(" . "); got != "." {
		t.Fatalf("got %q", got)
	}
	if got := SanitizarNombre(""); got != "" {
		t.Fatalf("got %q", got)
	}
}
