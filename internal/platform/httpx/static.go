package httpx

import (
	"net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/platform/config"
)

const panelCSP = "default-src 'self'; img-src 'self' data: blob:; script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; font-src 'self' data:; connect-src 'self'; " +
	"frame-ancestors 'none'; base-uri 'self'; form-action 'self'"

var hashedAsset = regexp.MustCompile(`(?i)[.-][0-9a-f]{8,}\.`)

// RegisterStatic installs the API 404 guard before the static/SPA fallback.
// Call it after every canonical and legacy API route has been registered.
func RegisterStatic(app *fiber.App, cfg *config.Config) {
	app.Use(traversalGuard())
	app.Use("/api", APINotFound)
	if cfg == nil {
		return
	}
	root, err := filepath.Abs(cfg.PublicDir)
	if err != nil {
		root = filepath.Clean(cfg.PublicDir)
	}
	app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		return c.Redirect("/shared/brand/favicon.svg", fiber.StatusFound)
	})
	app.Use(func(c *fiber.Ctx) error {
		if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
			return c.Next()
		}
		return serveStatic(c, cfg, root)
	})
}

// APINotFound guarantees unknown API routes never reach the SPA fallback.
func APINotFound(c *fiber.Ctx) error {
	return notFound(c)
}

func serveStatic(c *fiber.Ctx, cfg *config.Config, root string) error {
	requestPath := c.Path()
	if requestPath == "/robots.txt" || requestPath == "/shared" ||
		strings.HasPrefix(requestPath, "/shared/") {
		return serveAt(c, root, requestPath, false, true)
	}

	target := resolveTarget(c, cfg)
	if !target.ok {
		if requestPath == "/" && cfg.HTTP.HostRoutingMode != "host" &&
			cfg.HTTP.DefaultTenant == "none" {
			return serveAt(c, root, "/index.html", false, false)
		}
		return notFound(c)
	}
	c.Locals(tenantKey, target.tenant)
	return serveAt(c, filepath.Join(root, target.tenant), target.path, true, false)
}

func serveAt(c *fiber.Ctx, base, requestPath string, tenant, shared bool) error {
	rel := strings.TrimPrefix(pathpkg.Clean("/"+requestPath), "/")
	full := filepath.Join(base, filepath.FromSlash(rel))
	if !within(base, full) {
		return forbidden(c)
	}
	info, err := os.Stat(full)
	if err == nil && info.IsDir() {
		full = filepath.Join(full, "index.html")
		info, err = os.Stat(full)
	}
	if err == nil && !info.IsDir() {
		return sendFile(c, full, rel, shared)
	}
	if filepath.Ext(rel) != "" || !tenant {
		return notFound(c)
	}

	index := filepath.Join(base, "index.html")
	if info, err := os.Stat(index); err != nil || info.IsDir() {
		return notFound(c)
	}
	return sendFile(c, index, "index.html", false)
}

func sendFile(c *fiber.Ctx, full, rel string, shared bool) error {
	data, err := os.ReadFile(full)
	if err != nil {
		return notFound(c)
	}
	c.Set("Cache-Control", cacheFor(rel, shared))
	c.Type(filepath.Ext(full))
	if strings.HasSuffix(strings.ToLower(full), ".html") {
		c.Set("Content-Security-Policy", panelCSP)
	}
	return c.Send(data)
}

func cacheFor(rel string, shared bool) string {
	name := strings.ToLower(filepath.Base(rel))
	if shared {
		return "public, max-age=3600"
	}
	if name == "index.html" || name == "sw.js" || name == "manifest.json" {
		return "no-cache"
	}
	if hashedAsset.MatchString(name) {
		return "public, max-age=31536000, immutable"
	}
	return "no-cache"
}

func within(base, full string) bool {
	rel, err := filepath.Rel(base, full)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func traversalGuard() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if hasTraversal(c.OriginalURL()) || hasTraversal(c.Path()) {
			return forbidden(c)
		}
		return c.Next()
	}
}

func hasTraversal(raw string) bool {
	raw = strings.SplitN(raw, "?", 2)[0]
	decoded, err := url.PathUnescape(raw)
	if err != nil || strings.ContainsRune(decoded, '\x00') || strings.Contains(decoded, "\\") {
		return true
	}
	for _, segment := range strings.Split(decoded, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

func forbidden(c *fiber.Ctx) error {
	c.Set("Cache-Control", "no-store")
	return Fail(c, fiber.StatusForbidden, "FORBIDDEN", "ruta fuera de PUBLIC_DIR")
}

func notFound(c *fiber.Ctx) error {
	c.Set("Cache-Control", "no-store")
	return Fail(c, fiber.StatusNotFound, "NOT_FOUND", "recurso no encontrado")
}
