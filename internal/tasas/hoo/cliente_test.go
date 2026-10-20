package hoo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// servir levanta un origen de mentira que responde siempre el estado y el cuerpo
// indicados, para probar el cliente sin depender de que HOO esté en pie.
func servir(t *testing.T, estado int, cuerpo string) *httptest.Server {
	t.Helper()
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(estado)
		_, _ = w.Write([]byte(cuerpo))
	}))
	t.Cleanup(servidor.Close)
	return servidor
}

func TestObtenerMapeaElBorrador(t *testing.T) {
	servidor := servir(t, http.StatusOK, `{"id":7,"ars":123.45,"bob":"4050","usd":0,"created_at":"2026-10-18T01:02:03.000Z"}`)
	borrador, err := New(servidor.URL).Obtener(context.Background())
	if err != nil {
		t.Fatalf("Obtener falló: %v", err)
	}
	if borrador.ID != "7" || borrador.ARS != "123.45" || borrador.COP != "0" {
		t.Fatalf("los textos no se mapearon como el original: %+v", borrador)
	}
	if borrador.BOB == nil || *borrador.BOB != 4050 || borrador.PYUSD == nil || *borrador.PYUSD != 1.2 {
		t.Fatalf("los números o sus valores por defecto no coinciden: %+v", borrador)
	}
	if borrador.USD == nil || *borrador.USD != 1 {
		t.Fatalf("un usd en cero debe caer al valor por defecto 1: %+v", borrador.USD)
	}
	if borrador.CREATEDAT != "2026-10-18T01:02:03.000Z" {
		t.Fatalf("el sello del origen debe conservarse: %s", borrador.CREATEDAT)
	}
}

func TestObtenerAceptaUnArray(t *testing.T) {
	servidor := servir(t, http.StatusOK, `[{"id":"9","cop":"4.050,25"}]`)
	borrador, err := New(servidor.URL).Obtener(context.Background())
	if err != nil || borrador.ID != "9" || borrador.COP != "4.050,25" {
		t.Fatalf("se esperaba el primer elemento del array, llegó %+v con %v", borrador, err)
	}
}

func TestObtenerPropagaElEstadoHTTP(t *testing.T) {
	servidor := servir(t, http.StatusBadGateway, "<html>sin tasas</html>")
	_, err := New(servidor.URL).Obtener(context.Background())
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("se esperaba el estado 502 en el error, llegó %v", err)
	}
}

func TestMapearPublicaNullSiElTextoNoEsNumero(t *testing.T) {
	borrador, err := Mapear([]byte(`{"bob":"abc","pyg":""}`))
	if err != nil {
		t.Fatalf("Mapear no debía fallar: %v", err)
	}
	if borrador.BOB != nil || borrador.PYG == nil || *borrador.PYG != 0 {
		t.Fatalf("un texto no numérico es null y uno vacío cae a 0: %+v y %+v", borrador.BOB, borrador.PYG)
	}
}

func TestMapearRechazaUnCuerpoIlegible(t *testing.T) {
	if _, err := Mapear([]byte(`no soy json`)); err == nil {
		t.Fatal("se esperaba un error de decodificación")
	}
}

func TestMapearLaRespuestaRealDeHoo(t *testing.T) {
	// Payload capturado del origen real: es un array de un elemento y solo trae trece
	// divisas, así que las cinco restantes deben salir con su valor por defecto. Es la
	// definición ejecutable de "el puerto responde igual que el controlador Node".
	real := `[{"id":"83600","created_at":"2026-10-19T02:35:35.030Z","pen":3.45,"cop":3162,"clp":993,"ars":1618,"mxn":18.56,"ves":1006,"pyg":5765,"dop":62.4,"crc":455,"eur":0.9,"cad":1.45,"bob":11.85,"brl":5.06}]`
	borrador, err := Mapear([]byte(real))
	if err != nil {
		t.Fatalf("Mapear falló con el payload real: %v", err)
	}
	datos, err := json.Marshal(borrador)
	if err != nil {
		t.Fatalf("el borrador no se pudo serializar: %v", err)
	}
	esperado := `{"ID":"83600","ARS":"1618","BOB":11.85,"BRL":5.06,"CAD":1.45,"CLP":"993","COP":"3162","CRC":455,"DOP":62.4,"EUR":0.9,"MXN":18.56,"PEN":3.45,"PYG":5765,"VES":"1006","USD":1,"USDT":1,"PYUSD":1.2,"ECU":1,"PAN":1,"CREATED_AT":"2026-10-19T02:35:35.030Z"}`
	if string(datos) != esperado {
		t.Fatalf("el mapeo del payload real cambió:\n%s", datos)
	}
}
