package main

import (
	"os"
	"testing"
)

// TestRouteManifest expone la superficie registrada para check-routes.ps1.
// Solo imprime cuando el script lo pide, de modo que la suite normal no se
// ensucie con cien líneas de infraestructura.
func TestRouteManifest(t *testing.T) {
	if os.Getenv("GLAUK_ROUTE_MANIFEST") != "1" {
		t.Skip("manifiesto desactivado; lo activa scripts/check-routes.ps1")
	}
	app := testApp(t)
	for _, route := range app.GetRoutes(true) {
		if route.Method == "HEAD" || route.Method == "USE" {
			continue
		}
		t.Logf("ROUTE|%s|%s", route.Method, route.Path)
	}
}
