package httpx

import (
	"net"
	"strings"

	"github.com/gofiber/fiber/v2"

	"glauk/internal/platform/config"
)

type staticTarget struct {
	tenant string
	path   string
	ok     bool
}

// resolveTenant applies host-first resolution and then path prefixes. It is
// used by the static handler and by composition tests.
func resolveTenant(c *fiber.Ctx, cfg *config.Config) (string, bool) {
	target := resolveTarget(c, cfg)
	return target.tenant, target.ok
}

func resolveTarget(c *fiber.Ctx, cfg *config.Config) staticTarget {
	if cfg == nil {
		return staticTarget{}
	}
	mode := cfg.HTTP.HostRoutingMode
	if mode != "path" {
		if tenant := tenantForHost(requestHost(c, cfg), cfg); tenant != "" {
			return staticTarget{tenant: tenant, path: c.Path(), ok: true}
		}
	}
	if mode == "host" {
		return staticTarget{}
	}
	if tenant, rest, ok := tenantForPath(c.Path()); ok {
		return staticTarget{tenant: tenant, path: rest, ok: true}
	}
	if cfg.HTTP.DefaultTenant != "none" && cfg.HTTP.DefaultTenant != "" {
		return staticTarget{tenant: cfg.HTTP.DefaultTenant, path: c.Path(), ok: true}
	}
	return staticTarget{}
}

func requestHost(c *fiber.Ctx, cfg *config.Config) string {
	raw := c.Hostname()
	if header := strings.TrimSpace(cfg.HTTP.TrustedProxy); header != "" {
		if forwarded := c.Get(header); forwarded != "" {
			raw = forwarded
		}
	}
	return normalizeHost(raw)
}

func normalizeHost(raw string) string {
	raw = strings.TrimSpace(strings.Split(raw, ",")[0])
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	}
	return strings.ToLower(strings.TrimSuffix(raw, "."))
}

func tenantForHost(host string, cfg *config.Config) string {
	candidates := []struct {
		tenant string
		hosts  []string
	}{
		{"tasas", cfg.Tenants.Tasas},
		{"perfiles", cfg.Tenants.Perfiles},
		{"comprobantes", cfg.Tenants.Comprobantes},
		{"reportes", cfg.Tenants.Reportes},
	}
	for _, candidate := range candidates {
		for _, pattern := range candidate.hosts {
			if hostMatches(normalizeHost(pattern), host) {
				return candidate.tenant
			}
		}
	}
	return ""
}

func hostMatches(pattern, host string) bool {
	if pattern == "" || host == "" {
		return false
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:]
		if !strings.HasSuffix(host, suffix) {
			return false
		}
		label := strings.TrimSuffix(host, suffix)
		return label != "" && !strings.Contains(label, ".")
	}
	return pattern == host
}

func tenantForPath(requestPath string) (string, string, bool) {
	rest := strings.TrimPrefix(requestPath, "/")
	segment, tail, _ := strings.Cut(rest, "/")
	tenant := ""
	switch segment {
	case "tasas", "perfiles", "comprobantes", "reportes":
		tenant = segment
	default:
		return "", "", false
	}
	return tenant, "/" + tail, true
}
