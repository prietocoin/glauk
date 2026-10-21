// Package main is the composition root for the Glauk monolith.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"glauk/internal/platform/config"
	"glauk/internal/platform/evolution"
	"glauk/internal/platform/logging"
	"glauk/internal/platform/postgres"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "glauk: %v\n", err)
		os.Exit(1)
	}
}

// run sigue la secuencia de §2.9: configuración, logger, señales, pool,
// Evolution, cableado, rutas, escucha y apagado acotado. Ningún paquete de
// dominio abre puertos ni lee el entorno por su cuenta.
func run(parent context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cargar configuración: %w", err)
	}
	log := logging.NewLogger(cfg)

	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.New(ctx, cfg.DB)
	if err != nil {
		return fmt.Errorf("abrir PostgreSQL: %w", err)
	}
	defer pool.Close()

	evo := evolution.New(cfg.Evolution)
	deps, err := wire(pool, evo, cfg, log)
	if err != nil {
		return fmt.Errorf("cablear dominios: %w", err)
	}

	app := newApp(cfg, log)
	if err := registerRoutes(app, deps); err != nil {
		return fmt.Errorf("registrar rutas: %w", err)
	}

	errCh := startListening(app, listenAddress(cfg.Port))
	log.Info("glauk iniciado", "port", cfg.Port,
		"evolution_simulated", evo.Simulated())

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("servidor HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
		log.Info("señal de apagado recibida")
	}

	if err := shutdown(app, cfg.ShutdownTimeout); err != nil {
		return fmt.Errorf("apagar servidor: %w", err)
	}
	return nil
}
