package main

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/platform/config"
	"glauk/internal/platform/httpx"
)

// newApp construye la unica aplicacion Fiber con los middlewares globales.
// No registra rutas de dominio ni abre puertos: eso ocurre en run.
func newApp(cfg *config.Config, log *slog.Logger) *fiber.App {
	return httpx.NewApp(cfg, log)
}

// listenAddress conserva el puerto configurable en todas las interfaces, que
// es el contrato del contenedor. Un puerto invalido cae en 3000 para no
// arrancar en un puerto efimero por accidente.
func listenAddress(port int) string {
	if port <= 0 {
		port = 3000
	}
	return ":" + strconv.Itoa(port)
}

// startListening arranca Fiber en una goroutine y devuelve el canal por el que
// run observa el error de Listen. El canal tiene buffer 1 para que la goroutine
// pueda terminar aunque el apagado ya este en curso.
func startListening(app *fiber.App, address string) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Listen(address)
	}()
	return errCh
}

// shutdown acota el apagado con el timeout de configuracion. Context.Background
// es intencional: la senal que disparo el apagado ya cancelo el contexto raiz.
func shutdown(app *fiber.App, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return app.ShutdownWithContext(ctx)
}
