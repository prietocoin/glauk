package config

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func mapLookup(values map[string]string) lookupFunc {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func loadFrom(values map[string]string, appEnv string) (*Config, Errors) {
	cfg, errs := build(mapLookup(values), appEnv)
	if err := cfg.validate(); err != nil {
		if list, ok := err.(Errors); ok {
			errs = append(errs, list...)
		}
	}
	return cfg, errs
}

func TestDatabaseURLPrecedence(t *testing.T) {
	raw := "postgres://user:pass@db.example:5432/glauk?sslmode=require"
	cfg, errs := loadFrom(map[string]string{
		"DATABASE_URL": raw,
		"DB_HOST":      "ignored",
		"DB_USER":      "ignored",
	}, "development")
	if len(errs) > 0 {
		t.Fatalf("errores inesperados: %v", errs)
	}
	if got := cfg.DB.DSN(); got != raw {
		t.Fatalf("DSN = %q, se esperaba DATABASE_URL", got)
	}
}

func TestDSNComposesEscapedCredentials(t *testing.T) {
	db := DBConfig{
		User:     "user@corp",
		Password: "p@ss:word/with",
		Host:     "localhost",
		Port:     5432,
		Name:     "glauk",
		SSLMode:  "disable",
	}
	parsed, err := url.Parse(db.DSN())
	if err != nil {
		t.Fatalf("DSN inválido: %v", err)
	}
	if parsed.User.Username() != "user@corp" {
		t.Fatalf("usuario no escapado: %q", parsed.User.Username())
	}
	password, _ := parsed.User.Password()
	if password != "p@ss:word/with" {
		t.Fatalf("contraseña no escapada: %q", password)
	}
	if parsed.Query().Get("sslmode") != "disable" {
		t.Fatalf("sslmode perdido: %q", db.DSN())
	}
}

func TestValidateDSN(t *testing.T) {
	cases := []struct {
		name string
		dsn  string
		ok   bool
	}{
		{"válido", "postgres://u:p@localhost:5432/glauk", true},
		{"postgresql", "postgresql://u:p@localhost/glauk", true},
		{"esquema", "mysql://u:p@localhost/glauk", false},
		{"sin host", "postgres:///glauk", false},
		{"sin base", "postgres://localhost:5432", false},
	}
	for _, tc := range cases {
		err := ValidateDSN(tc.dsn)
		if (err == nil) != tc.ok {
			t.Errorf("%s: err = %v, ok esperado %v", tc.name, err, tc.ok)
		}
	}
}

func TestProductionRequiresDatabaseAndAggregates(t *testing.T) {
	_, errs := loadFrom(map[string]string{"APP_ENV": "production", "PORT": "0"}, "production")
	if len(errs) < 2 {
		t.Fatalf("se esperaban errores agregados, got %v", errs)
	}
	message := errs.Error()
	if !strings.Contains(message, "configuración inválida") {
		t.Fatalf("falta cabecera agregada: %s", message)
	}
	if !strings.Contains(message, "DATABASE_URL") {
		t.Fatalf("falta error de base: %s", message)
	}
	if !strings.Contains(message, "PORT") {
		t.Fatalf("falta error de puerto: %s", message)
	}
}

func TestEvolutionURLAndKeyMustTravelTogether(t *testing.T) {
	_, errs := loadFrom(map[string]string{"EVOLUTION_API_URL": "https://evo.example"}, "development")
	if !strings.Contains(errs.Error(), "EVOLUTION_API_KEY") {
		t.Fatalf("se esperaba error de clave: %v", errs)
	}

	cfg, errs := loadFrom(map[string]string{
		"EVOLUTION_API_URL": "https://evo.example/",
		"EVOLUTION_API_KEY": "canonical-secret",
	}, "development")
	if len(errs) > 0 {
		t.Fatalf("errores inesperados: %v", errs)
	}
	if cfg.Evolution.BaseURL != "https://evo.example" {
		t.Fatalf("URL no normalizada: %q", cfg.Evolution.BaseURL)
	}
	if cfg.Evolution.Instance != "Glauk" {
		t.Fatalf("instancia = %q", cfg.Evolution.Instance)
	}
}

func TestEvolutionKeysConflictWithoutLeaking(t *testing.T) {
	_, errs := loadFrom(map[string]string{
		"EVOLUTION_API_URL":      "https://evo.example",
		"EVOLUTION_API_KEY":      "secret-one",
		"AUTHENTICATION_API_KEY": "secret-two",
	}, "development")
	message := errs.Error()
	if !strings.Contains(message, "AUTHENTICATION_API_KEY") {
		t.Fatalf("falta nombre de variable: %s", message)
	}
	if strings.Contains(message, "secret-one") || strings.Contains(message, "secret-two") {
		t.Fatalf("se filtró una clave: %s", message)
	}
}

func TestPortAndPoolRanges(t *testing.T) {
	cases := []struct {
		values map[string]string
		want   string
	}{
		{map[string]string{"PORT": "0"}, "PORT"},
		{map[string]string{"PORT": "70000"}, "PORT"},
		{map[string]string{"DB_MAX_CONNS": "2", "DB_MIN_CONNS": "3"}, "DB_MIN_CONNS"},
		{map[string]string{"DB_MAX_CONNS": "2147483648"}, "DB_MAX_CONNS"},
	}
	for _, tc := range cases {
		_, errs := loadFrom(tc.values, "development")
		if !strings.Contains(errs.Error(), tc.want) {
			t.Errorf("%v: se esperaba %q, got %v", tc.values, tc.want, errs)
		}
	}
}

func TestByteUnitsAreEquivalent(t *testing.T) {
	const expected = int64(20 << 20)
	for _, raw := range []string{"20MiB", "20MB", "20971520"} {
		value, err := envBytes(mapLookup(map[string]string{"SIZE": raw}), "SIZE", 0)
		if err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		if value != expected {
			t.Fatalf("%s = %d, esperado %d", raw, value, expected)
		}
	}
}

func TestDurationAndCSVParsing(t *testing.T) {
	value, err := envDuration(mapLookup(map[string]string{"T": "25s"}), "T", time.Second)
	if err != nil || value != 25*time.Second {
		t.Fatalf("duración = %v, err = %v", value, err)
	}
	if _, err := envDuration(mapLookup(map[string]string{"T": "0s"}), "T", time.Second); err == nil {
		t.Fatal("0s debe fallar")
	}
	origins := envCSV(mapLookup(map[string]string{"O": "https://a, https://b ,"}), "O", []string{"*"})
	if len(origins) != 2 || origins[0] != "https://a" || origins[1] != "https://b" {
		t.Fatalf("CSV = %#v", origins)
	}
}

func TestValidProductionConfiguration(t *testing.T) {
	cfg, errs := loadFrom(map[string]string{
		"DATABASE_URL":         "postgres://glauk:secret@db:5432/glauk?sslmode=require",
		"EVOLUTION_API_URL":    "https://evo.example",
		"EVOLUTION_API_KEY":    "canonical-secret",
		"CORS_ALLOWED_ORIGINS": "https://app.example",
		"JID_GRUPO_PRUEBA":     "123456789@g.us",
	}, "production")
	if len(errs) > 0 {
		t.Fatalf("production válida rechazada: %v", errs)
	}
	if cfg.Tenants.JIDGrupoPrueba != "123456789@g.us" {
		t.Fatalf("JID_GRUPO_PRUEBA no cargado: %q", cfg.Tenants.JIDGrupoPrueba)
	}
}
