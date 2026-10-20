package config

import (
	"fmt"
	"net/url"
	"strconv"
)

// DSN returns DATABASE_URL when present; otherwise it composes an escaped
// PostgreSQL URL from the DB_* components.
func (c DBConfig) DSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   c.Host + ":" + strconv.Itoa(c.Port),
		Path:   "/" + c.Name,
	}
	q := u.Query()
	q.Set("sslmode", c.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

// ValidateDSN checks scheme, host, and database name without echoing
// credentials or the full connection string.
func ValidateDSN(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("DATABASE_URL inválida: formato no válido")
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return fmt.Errorf("DATABASE_URL debe usar esquema postgres o postgresql")
	}
	if u.Host == "" {
		return fmt.Errorf("DATABASE_URL debe incluir host")
	}
	if u.Path == "" || u.Path == "/" {
		return fmt.Errorf("DATABASE_URL debe incluir el nombre de la base")
	}
	return nil
}
