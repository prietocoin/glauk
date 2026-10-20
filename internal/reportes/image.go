package reportes

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg" // registra el decodificador JPEG
	"strings"
)

const jpegMagic = "\xff\xd8\xff"

// Imagen es un JPEG validado y normalizado a base64 estándar.
type Imagen struct {
	Bytes  []byte
	Base64 string
	Ancho  int
	Alto   int
}

// DecodificarImagen valida firma, tamaño y dimensiones usando DecodeConfig.
// No descomprime el bitmap completo: solo lee la cabecera del JPEG.
func DecodificarImagen(raw string, cfg Config) (Imagen, error) {
	cfg = cfg.withDefaults()
	payload, err := LimpiarDataURL(raw)
	if err != nil {
		return Imagen{}, err
	}
	if payload == "" {
		return Imagen{}, ErrImagenRequerida
	}

	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(payload)
		if err != nil {
			return Imagen{}, fmt.Errorf("%w: base64 inválido", ErrImagenFormato)
		}
	}
	if int64(len(decoded)) > cfg.ImageMaxBytes {
		return Imagen{}, fmt.Errorf("%w: %d > %d bytes",
			ErrImagenMuyGrande, len(decoded), cfg.ImageMaxBytes)
	}
	if len(decoded) < 4 || !bytes.HasPrefix(decoded, []byte(jpegMagic)) {
		return Imagen{}, fmt.Errorf("%w: firma binaria ausente", ErrImagenFormato)
	}

	conf, format, err := image.DecodeConfig(bytes.NewReader(decoded))
	if err != nil || format != "jpeg" {
		return Imagen{}, fmt.Errorf("%w: no se pudo decodificar", ErrImagenFormato)
	}
	if conf.Width < cfg.ImageMinDimension || conf.Height < cfg.ImageMinDimension {
		return Imagen{}, fmt.Errorf("%w: %dx%d", ErrImagenDimensiones, conf.Width, conf.Height)
	}
	if conf.Width > cfg.ImageMaxDimension || conf.Height > cfg.ImageMaxDimension {
		return Imagen{}, fmt.Errorf("%w: %dx%d", ErrImagenDimensiones, conf.Width, conf.Height)
	}
	if int64(conf.Width)*int64(conf.Height) > cfg.ImageMaxPixels {
		return Imagen{}, fmt.Errorf("%w: %d píxeles", ErrImagenDimensiones, conf.Width*conf.Height)
	}

	return Imagen{
		Bytes:  decoded,
		Base64: base64.StdEncoding.EncodeToString(decoded),
		Ancho:  conf.Width,
		Alto:   conf.Height,
	}, nil
}

// LimpiarDataURL acepta "data:image/jpeg;base64,..." y también base64 puro.
// Devuelve ErrImagenFormato si el data URL declara otro MIME o no es base64.
func LimpiarDataURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(strings.ToLower(trimmed), "data:") {
		return trimmed, nil
	}
	comma := strings.IndexByte(trimmed, ',')
	if comma < 0 {
		return "", fmt.Errorf("%w: data URL sin coma", ErrImagenFormato)
	}
	header := strings.ToLower(trimmed[:comma])
	if !strings.Contains(header, "image/jpeg") && !strings.Contains(header, "image/jpg") {
		return "", fmt.Errorf("%w: MIME no soportado", ErrImagenFormato)
	}
	if !strings.Contains(header, ";base64") {
		return "", fmt.Errorf("%w: data URL no base64", ErrImagenFormato)
	}
	return strings.TrimSpace(trimmed[comma+1:]), nil
}

// SanitizarNombre elimina rutas y caracteres de control y limita a 120 runas.
// No inventa extensiones ni nombres: si el cliente no envía fileName, queda "".
func SanitizarNombre(raw string) string {
	name := strings.ReplaceAll(strings.TrimSpace(raw), `\`, "/")
	if slash := strings.LastIndexByte(name, '/'); slash >= 0 {
		name = name[slash+1:]
	}
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	runes := []rune(name)
	if len(runes) > 120 {
		name = string(runes[:120])
	}
	return name
}
