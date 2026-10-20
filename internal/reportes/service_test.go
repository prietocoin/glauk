package reportes

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeSocios struct {
	socio Socio
	err   error
	calls int
}

func (f *fakeSocios) SocioPorNombre(ctx context.Context, nombre string) (Socio, error) {
	f.calls++
	if f.err != nil {
		return Socio{}, f.err
	}
	return f.socio, nil
}

func (f *fakeSocios) Socios(ctx context.Context, rol string) ([]Socio, error) {
	return []Socio{f.socio}, f.err
}

type fakeMovimientos struct {
	filtros Filtros
	err     error
	rol     string
}

func (f *fakeMovimientos) Filtros(ctx context.Context, rol string) (Filtros, error) {
	f.rol = rol
	return f.filtros, f.err
}

func (f *fakeMovimientos) Movimientos(ctx context.Context, q ConsultaMovimientos) ([]Movimiento, error) {
	return nil, f.err
}

func (f *fakeMovimientos) Resumen(ctx context.Context, q ConsultaMovimientos) (Resumen, error) {
	return Resumen{}, f.err
}

type fakeSender struct {
	imagen *MensajeImagen
	texto  *MensajeTexto
	err    error
	calls  int
}

func (f *fakeSender) EnviarImagen(ctx context.Context, msg MensajeImagen) (Recibo, error) {
	f.calls++
	copy := msg
	f.imagen = &copy
	if f.err != nil {
		return Recibo{}, f.err
	}
	return Recibo{MessageID: "img-1", Estado: "enviado"}, nil
}

func (f *fakeSender) EnviarTexto(ctx context.Context, msg MensajeTexto) (Recibo, error) {
	f.calls++
	copy := msg
	f.texto = &copy
	if f.err != nil {
		return Recibo{}, f.err
	}
	return Recibo{MessageID: "txt-1", Estado: "enviado"}, nil
}

func testService(socios *fakeSocios, movimientos *fakeMovimientos, sender *fakeSender) *Service {
	return NewService(socios, movimientos, sender, Config{ImageMaxBytes: 4 << 20})
}

func TestEnviarTextoRemoteJIDGanaSobreSocio(t *testing.T) {
	socios := &fakeSocios{socio: Socio{Nombre: "FUNDA", IDGrupo: "99999@s.whatsapp.net"}}
	sender := &fakeSender{}
	svc := testService(socios, &fakeMovimientos{}, sender)

	res, err := svc.EnviarTexto(context.Background(), SolicitudTexto{
		Socio: "FUNDA", RemoteJID: "12345@c.us", Texto: "  hola  ",
	})
	if err != nil {
		t.Fatalf("EnviarTexto: %v", err)
	}
	if socios.calls != 0 {
		t.Fatalf("se consultó el socio %d veces; remoteJid debe ganar", socios.calls)
	}
	if res.JID != "12345@s.whatsapp.net" {
		t.Fatalf("JID = %q", res.JID)
	}
	if sender.texto == nil || sender.texto.Destino != res.JID || sender.texto.Texto != "hola" {
		t.Fatalf("mensaje enviado = %+v", sender.texto)
	}
}

func TestEnviarTextoResuelveSocio(t *testing.T) {
	socios := &fakeSocios{socio: Socio{Nombre: "FUNDA", IDGrupo: "12345@s.whatsapp.net"}}
	sender := &fakeSender{}
	svc := testService(socios, &fakeMovimientos{}, sender)

	if _, err := svc.EnviarTexto(context.Background(), SolicitudTexto{Socio: " FUNDA ", Texto: "ok"}); err != nil {
		t.Fatalf("EnviarTexto: %v", err)
	}
	if socios.calls != 1 || sender.texto == nil || sender.texto.Destino != "12345@s.whatsapp.net" {
		t.Fatalf("calls=%d mensaje=%+v", socios.calls, sender.texto)
	}
}

func TestResolverDestinoErrores(t *testing.T) {
	cases := []struct {
		name   string
		socios *fakeSocios
		in     SolicitudTexto
		want   error
	}{
		{"sin destino", &fakeSocios{}, SolicitudTexto{Texto: "hola"}, ErrDestinoRequerido},
		{"socio inexistente", &fakeSocios{err: ErrSocioNoEncontrado}, SolicitudTexto{Socio: "X", Texto: "hola"}, ErrSocioNoEncontrado},
		{"socio sin jid", &fakeSocios{socio: Socio{Nombre: "X"}}, SolicitudTexto{Socio: "X", Texto: "hola"}, ErrJIDNoDisponible},
		{"placeholder", &fakeSocios{socio: Socio{Nombre: "X", IDGrupo: "X_JID"}}, SolicitudTexto{Socio: "X", Texto: "hola"}, ErrJIDNoDisponible},
		{"remote inválido", &fakeSocios{socio: Socio{Nombre: "X", IDGrupo: "12345@s.whatsapp.net"}}, SolicitudTexto{RemoteJID: "foo@g.us", Texto: "hola"}, ErrJIDInvalido},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := testService(tc.socios, &fakeMovimientos{}, &fakeSender{})
			_, err := svc.EnviarTexto(context.Background(), tc.in)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, quiero %v", err, tc.want)
			}
		})
	}
}

func TestEnviarImagenValida(t *testing.T) {
	sender := &fakeSender{}
	svc := testService(&fakeSocios{}, &fakeMovimientos{}, sender)

	res, err := svc.EnviarImagen(context.Background(), SolicitudImagen{
		RemoteJID: "12345@g.us", Base64: jpegBase64(t, 240, 240),
		Caption: "  reporte  ", FileName: `..\..\reporte.jpg`,
	})
	if err != nil {
		t.Fatalf("EnviarImagen: %v", err)
	}
	if res.JID != "12345@g.us" || res.MessageID != "img-1" || res.Bytes == 0 {
		t.Fatalf("resultado = %+v", res)
	}
	if sender.imagen == nil {
		t.Fatal("no se envió imagen")
	}
	if sender.imagen.MimeType != "image/jpeg" || sender.imagen.Caption != "reporte" || sender.imagen.FileName != "reporte.jpg" {
		t.Fatalf("mensaje = %+v", sender.imagen)
	}
}

func TestEnviarImagenRequerida(t *testing.T) {
	svc := testService(&fakeSocios{}, &fakeMovimientos{}, &fakeSender{})
	_, err := svc.EnviarImagen(context.Background(), SolicitudImagen{RemoteJID: "12345@g.us"})
	if !errors.Is(err, ErrImagenRequerida) {
		t.Fatalf("err = %v", err)
	}
}

func TestEnviarTextoRequerido(t *testing.T) {
	svc := testService(&fakeSocios{}, &fakeMovimientos{}, &fakeSender{})
	_, err := svc.EnviarTexto(context.Background(), SolicitudTexto{RemoteJID: "12345@g.us", Texto: "  "})
	if !errors.Is(err, ErrTextoRequerido) {
		t.Fatalf("err = %v", err)
	}
}

func TestEnviarFallidoEnvuelveError(t *testing.T) {
	sender := &fakeSender{err: errors.New("evolution caído")}
	svc := testService(&fakeSocios{}, &fakeMovimientos{}, sender)
	_, err := svc.EnviarTexto(context.Background(), SolicitudTexto{RemoteJID: "12345@g.us", Texto: "hola"})
	if !errors.Is(err, ErrEnvioFallido) || !errors.Is(err, sender.err) {
		t.Fatalf("err = %v", err)
	}
}

func TestEnviarTextoRecorta(t *testing.T) {
	sender := &fakeSender{}
	svc := testService(&fakeSocios{}, &fakeMovimientos{}, sender)
	if _, err := svc.EnviarTexto(context.Background(), SolicitudTexto{
		RemoteJID: "12345@g.us", Texto: strings.Repeat("a", 5000),
	}); err != nil {
		t.Fatalf("EnviarTexto: %v", err)
	}
	if sender.texto == nil {
		t.Fatal("no se envió texto")
	}
	if len(sender.texto.Texto) != 4096 {
		t.Fatalf("largo = %d", len(sender.texto.Texto))
	}
}

func TestFiltrosDelega(t *testing.T) {
	movs := &fakeMovimientos{filtros: Filtros{Roles: []string{"admin"}}}
	svc := testService(&fakeSocios{}, movs, &fakeSender{})
	got, err := svc.Filtros(context.Background(), "  admin ")
	if err != nil {
		t.Fatalf("Filtros: %v", err)
	}
	if movs.rol != "admin" || len(got.Roles) != 1 || got.Roles[0] != "admin" {
		t.Fatalf("rol=%q filtros=%+v", movs.rol, got)
	}
}
