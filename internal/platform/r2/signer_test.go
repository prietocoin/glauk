package r2

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
	"time"

	"glauk/internal/platform/config"
)

func signedRequest(t *testing.T, body []byte) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut,
		"https://acc.r2.cloudflarestorage.com/bucket/comprobantes/a.jpg",
		bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "image/jpeg")
	return req
}

func TestSignEsDeterministaYCambiaConElCuerpo(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	signer := signer{accessKey: "AK", secretKey: "SK"}
	body := []byte("imagen")

	first := signedRequest(t, body)
	signer.sign(first, body, now)
	second := signedRequest(t, body)
	signer.sign(second, body, now)

	auth := first.Header.Get("Authorization")
	if auth != second.Header.Get("Authorization") {
		t.Fatalf("firma no determinista:\n%s\n%s", auth, second.Header.Get("Authorization"))
	}
	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 Credential=AK/20260102/auto/s3/aws4_request") {
		t.Fatalf("authorization = %s", auth)
	}
	if first.Header.Get("x-amz-date") != "20260102T030405Z" {
		t.Fatalf("x-amz-date = %s", first.Header.Get("x-amz-date"))
	}
	if len(first.Header.Get("x-amz-content-sha256")) != 64 {
		t.Fatalf("payload hash = %s", first.Header.Get("x-amz-content-sha256"))
	}

	other := signedRequest(t, []byte("otra"))
	signer.sign(other, []byte("otra"), now)
	if other.Header.Get("Authorization") == auth {
		t.Fatal("cuerpos distintos no pueden compartir firma")
	}
}

func TestCanonicalURIEscapaElObjeto(t *testing.T) {
	client := New(config.R2Config{
		AccountID: "acc", AccessKeyID: "ak", SecretAccessKey: "sk",
		Bucket: "bucket", PublicDomain: "https://cdn.example/",
	})
	target, err := client.objectURL("comprobantes/a b.jpg")
	if err != nil {
		t.Fatalf("objectURL: %v", err)
	}
	if !strings.Contains(target.EscapedPath(), "a%20b.jpg") {
		t.Fatalf("path = %s", target.EscapedPath())
	}
	if client.PublicURL("/comprobantes/a.jpg") != "https://cdn.example/comprobantes/a.jpg" {
		t.Fatalf("public url = %s", client.PublicURL("comprobantes/a.jpg"))
	}
}

func TestNewDerivaEndpointYConfigured(t *testing.T) {
	client := New(config.R2Config{
		AccountID: "acc", AccessKeyID: "ak", SecretAccessKey: "sk",
		Bucket: "bucket",
	})
	if client.endpoint != "https://acc.r2.cloudflarestorage.com" {
		t.Fatalf("endpoint = %s", client.endpoint)
	}
	if !client.Configured() {
		t.Fatal("debería estar configurado")
	}
	empty := New(config.R2Config{})
	if empty.Configured() {
		t.Fatal("sin credenciales no está configurado")
	}
}
