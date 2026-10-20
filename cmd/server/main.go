// Package main is the composition root for the Glauk monolith.
//
// Phase 1 only exposes a liveness endpoint. Configuration, PostgreSQL,
// Evolution, and domain wiring are incorporated in later runbook phases.
package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

const (
	defaultAddress = ":3000"
	healthPath     = "/health"
)

// newApp builds the HTTP application without opening ports. Keeping the
// constructor separate makes the app testable and leaves room for the
// platform middleware added in phase 5.
func newApp() *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get(healthPath, health)
	return app
}

// health is a read-only liveness probe. It deliberately does not touch the
// database or Evolution; deeper checks arrive with /ready in phase 3.
func health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func main() {
	if err := newApp().Listen(defaultAddress); err != nil {
		log.Fatalf("glauk: servidor detenido: %v", err)
	}
}
