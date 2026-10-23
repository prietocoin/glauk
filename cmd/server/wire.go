package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	comprobanteshttp "glauk/internal/comprobantes/http"
	comprobantesservice "glauk/internal/comprobantes/service"
	comprobantesstore "glauk/internal/comprobantes/store"
	"glauk/internal/ingesta"
	perfileshttp "glauk/internal/perfiles/http"
	perfilesrepo "glauk/internal/perfiles/repo"
	perfilesservice "glauk/internal/perfiles/service"
	"glauk/internal/platform/config"
	"glauk/internal/platform/evolution"
	"glauk/internal/platform/httpx"
	"glauk/internal/reportes"
	reportesadapters "glauk/internal/reportes/adapters"
	reporteshttp "glauk/internal/reportes/http"
	tasashoo "glauk/internal/tasas/hoo"
	tasashttp "glauk/internal/tasas/http"
	tasasservice "glauk/internal/tasas/service"
)

// Dependencies reúne las piezas construidas una sola vez por wire. routes.go
// solo las conecta; ningún handler abre puertos ni construye servicios.
type Dependencies struct {
	Config       *config.Config
	Log          *slog.Logger
	Health       fiber.Handler
	Ready        fiber.Handler
	Tasas        *tasashttp.Deps
	Perfiles     perfileshttp.Deps
	Comprobantes comprobanteshttp.Deps
	Reportes     *reporteshttp.Handler
	Guard        fiber.Handler
	Ingesta      *ingesta.Module
}

// wire ensambla repositorios, servicios, adaptadores y handlers. Es la única
// función que conoce todos los dominios y el único lugar donde se cruzan
// sus dependencias (§4.3).
func wire(pool *pgxpool.Pool, evo *evolution.Client, cfg *config.Config,
	log *slog.Logger) (*Dependencies, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuración nil")
	}

	perfilesReader := perfilesrepo.NewReader(pool, log)
	perfilesWriter := perfilesrepo.NewWriter(pool)
	perfilesService := perfilesservice.New(perfilesReader, perfilesWriter, log)

	comprobantesRepo := comprobantesstore.New(pool)
	comprobantesService := comprobantesservice.New(comprobantesRepo)

	tasasService := tasasservice.New(pool, evo, cfg)
	tasasDeps := tasashttp.NewDeps(tasasService, tasashoo.New(cfg.HOO.URL), time.Now)

	reportesCfg := reportes.Config{
		ImageMaxBytes:     cfg.Report.ImageMaxBytes,
		ImageMinDimension: cfg.Report.ImageMinDim,
		ImageMaxDimension: cfg.Report.ImageMaxDim,
		ImageMaxPixels:    cfg.Report.ImageMaxPixels,
		IdempotencyTTL:    cfg.Report.IdempotencyTTL,
	}
	reportesModule := reportesadapters.New(perfilesReader, comprobantesService, evo, reportesCfg)
	reportesModule.RateLimiter = reportes.NewRateLimiter(cfg.Report.RatePerMinute)

	health, ready := newHealthHandlers(pool, evo, cfg.Evolution.Instance, time.Now())
	return &Dependencies{
		Config:       cfg,
		Log:          log,
		Health:       health,
		Ready:        ready,
		Tasas:        tasasDeps,
		Perfiles:     perfileshttp.NewDeps(perfilesService),
		Comprobantes: comprobanteshttp.NewDeps(comprobantesService, cfg),
		Reportes:     reporteshttp.NewHandler(reportesModule),
		Guard:        httpx.RequireAPIKey(cfg),
		Ingesta:      ingesta.NewModule(cfg, pool, evo, log),
	}, nil
}
