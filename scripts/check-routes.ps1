param([string]$Root)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($Root)) {
    $Root = Split-Path -Parent $PSScriptRoot
}

function New-Route([string]$Method, [string]$Path, [string]$Successor = '') {
    [pscustomobject]@{ Method = $Method; Path = $Path; Successor = $Successor }
}

# Matriz normativa de la Seccion 5. Solo se listan rutas de API y sondas.
$Canonical = @(
    New-Route 'GET' '/health'
    New-Route 'GET' '/ready'

    New-Route 'POST' '/api/v1/webhooks/evolution'

    New-Route 'GET' '/api/v1/tasas/monitor'
    New-Route 'GET' '/api/v1/tasas/socios'
    New-Route 'GET' '/api/v1/tasas/hoo-borrador'
    New-Route 'PUT' '/api/v1/tasas/socios/:socio'
    New-Route 'POST' '/api/v1/tasas/guardar-lote'
    New-Route 'POST' '/api/v1/tasas/publicar-lote'
    New-Route 'POST' '/api/v1/tasas/notificar-remesas'
    New-Route 'GET' '/api/v1/tasas/calcular/:socio'
    New-Route 'GET' '/api/v1/tasas/render/:socio'
    New-Route 'POST' '/api/v1/tasas/render'
    New-Route 'POST' '/api/v1/tasas/disparar'
    New-Route 'POST' '/api/v1/tasas/disparar-lote'
    New-Route 'DELETE' '/api/v1/tasas/lote/:id_tasa'

    New-Route 'GET' '/api/v1/perfiles'
    New-Route 'GET' '/api/v1/perfiles/activos'
    New-Route 'POST' '/api/v1/perfiles/guardar'
    New-Route 'POST' '/api/v1/perfiles/apagar-todos'
    New-Route 'POST' '/api/v1/perfiles/plantilla/memorizar'
    New-Route 'POST' '/api/v1/perfiles/plantilla/restaurar'
    New-Route 'PATCH' '/api/v1/perfiles/:nombre/estado'
    New-Route 'DELETE' '/api/v1/perfiles/:nombre'

    New-Route 'GET' '/api/v1/comprobantes'
    New-Route 'POST' '/api/v1/comprobantes/liquidar'
    New-Route 'POST' '/api/v1/comprobantes/:hashLargo/releer'
    New-Route 'PUT' '/api/v1/comprobantes/:hashLargo'
    New-Route 'DELETE' '/api/v1/comprobantes/:hashLargo'
    New-Route 'GET' '/api/v1/comprobantes/tasas/ultimas'
    New-Route 'GET' '/api/v1/comprobantes/tasas/historial'

    New-Route 'GET' '/api/v1/reportes/filtros'
    New-Route 'POST' '/api/v1/reportes/imagen'
    New-Route 'POST' '/api/v1/reportes/texto'

    New-Route 'GET' '/api/v1/admin/cola'
    New-Route 'PUT' '/api/v1/admin/cola/:hashLargo'
    New-Route 'DELETE' '/api/v1/admin/cola/:hashLargo'
)

$Aliases = @(
    New-Route 'GET' '/api/tasas/monitor' '/api/v1/tasas/monitor'
    New-Route 'GET' '/api/tasas/socios' '/api/v1/tasas/socios'
    New-Route 'GET' '/api/tasas/hoo-borrador' '/api/v1/tasas/hoo-borrador'
    New-Route 'PUT' '/api/tasas/socios/:socio' '/api/v1/tasas/socios/:socio'
    New-Route 'POST' '/api/tasas/guardar-lote' '/api/v1/tasas/guardar-lote'
    New-Route 'POST' '/api/tasas/publicar-lote' '/api/v1/tasas/publicar-lote'
    New-Route 'POST' '/api/tasas/notificar-remesas' '/api/v1/tasas/notificar-remesas'
    New-Route 'GET' '/api/tasas/calcular/:socio' '/api/v1/tasas/calcular/:socio'
    New-Route 'GET' '/api/tasas/render/:socio' '/api/v1/tasas/render/:socio'
    New-Route 'POST' '/api/tasas/render' '/api/v1/tasas/render'
    New-Route 'POST' '/api/tasas/disparar' '/api/v1/tasas/disparar'
    New-Route 'POST' '/api/tasas/disparar-lote' '/api/v1/tasas/disparar-lote'
    New-Route 'DELETE' '/api/tasas/lote/:id_tasa' '/api/v1/tasas/lote/:id_tasa'
    New-Route 'GET' '/api/tasas/fetch-hoo' '/api/v1/tasas/hoo-borrador'
    New-Route 'POST' '/api/tasas/publicar' '/api/v1/tasas/publicar-lote'
    New-Route 'POST' '/api/tasas/reenviar' '/api/v1/tasas/publicar-lote'
    New-Route 'GET' '/api/preview-image/:identificador' '/api/v1/tasas/render/:socio'
    New-Route 'GET' '/api/preview-image' '/api/v1/tasas/render/:socio'

    New-Route 'GET' '/api/directorio' '/api/v1/perfiles'
    New-Route 'DELETE' '/api/directorio/:nombre' '/api/v1/perfiles/:nombre'
    New-Route 'GET' '/api/socios' '/api/v1/perfiles'
    New-Route 'POST' '/api/socios/config' '/api/v1/perfiles/guardar'
    New-Route 'PATCH' '/api/socios/:nombre/estado' '/api/v1/perfiles/:nombre/estado'
    New-Route 'PATCH' '/api/socios/desactivar-todos' '/api/v1/perfiles/apagar-todos'
    New-Route 'POST' '/api/socios/guardar-vigentes' '/api/v1/perfiles/plantilla/memorizar'
    New-Route 'POST' '/api/socios/restaurar-vigentes' '/api/v1/perfiles/plantilla/restaurar'
    New-Route 'GET' '/api/v1/atenea/directorio' '/api/v1/perfiles'
    New-Route 'GET' '/api/v1/atenea/socios' '/api/v1/perfiles'
    New-Route 'POST' '/api/v1/atenea/socios/config' '/api/v1/perfiles/guardar'
    New-Route 'PATCH' '/api/v1/atenea/socios/:nombre/estado' '/api/v1/perfiles/:nombre/estado'
    New-Route 'PATCH' '/api/v1/atenea/socios/desactivar-todos' '/api/v1/perfiles/apagar-todos'
    New-Route 'POST' '/api/v1/atenea/socios/guardar-vigentes' '/api/v1/perfiles/plantilla/memorizar'
    New-Route 'POST' '/api/v1/atenea/socios/restaurar-vigentes' '/api/v1/perfiles/plantilla/restaurar'
    New-Route 'DELETE' '/api/v1/atenea/directorio/:nombre' '/api/v1/perfiles/:nombre'

    New-Route 'GET' '/api/comprobantes' '/api/v1/comprobantes'
    New-Route 'GET' '/api/cola' '/api/v1/comprobantes'
    New-Route 'GET' '/api/reportes' '/api/v1/comprobantes'
    New-Route 'GET' '/api/reportes/operaciones' '/api/v1/comprobantes'
    New-Route 'POST' '/api/comprobantes/liquidar' '/api/v1/comprobantes/liquidar'
    New-Route 'POST' '/api/comprobantes/:hashLargo/releer' '/api/v1/comprobantes/:hashLargo/releer'
    New-Route 'PUT' '/api/comprobantes/:hashLargo' '/api/v1/comprobantes/:hashLargo'
    New-Route 'DELETE' '/api/comprobantes/:hashLargo' '/api/v1/comprobantes/:hashLargo'
    New-Route 'GET' '/api/tasas/ultimas' '/api/v1/comprobantes/tasas/ultimas'
    New-Route 'GET' '/api/tasas/historial' '/api/v1/comprobantes/tasas/historial'
    New-Route 'GET' '/api/v1/atenea/comprobantes' '/api/v1/comprobantes'
    New-Route 'POST' '/api/v1/atenea/comprobantes/liquidar' '/api/v1/comprobantes/liquidar'
    New-Route 'POST' '/api/v1/atenea/comprobantes/:hashLargo/releer' '/api/v1/comprobantes/:hashLargo/releer'
    New-Route 'PUT' '/api/v1/atenea/comprobantes/:hashLargo' '/api/v1/comprobantes/:hashLargo'
    New-Route 'DELETE' '/api/v1/atenea/comprobantes/:hashLargo' '/api/v1/comprobantes/:hashLargo'
    New-Route 'GET' '/api/v1/atenea/cola' '/api/v1/comprobantes'
    New-Route 'GET' '/api/v1/atenea/reportes' '/api/v1/comprobantes'
    New-Route 'GET' '/api/v1/atenea/reportes/operaciones' '/api/v1/comprobantes'
    New-Route 'GET' '/api/v1/atenea/tasas/ultimas' '/api/v1/comprobantes/tasas/ultimas'
    New-Route 'GET' '/api/v1/atenea/tasas/historial' '/api/v1/comprobantes/tasas/historial'

    New-Route 'GET' '/api/reportes/filtros' '/api/v1/reportes/filtros'
    New-Route 'POST' '/api/reportes/enviar-whatsapp' '/api/v1/reportes/imagen'
    New-Route 'POST' '/api/whatsapp/enviar-media' '/api/v1/reportes/imagen'
    New-Route 'GET' '/api/v1/atenea/reportes/filtros' '/api/v1/reportes/filtros'
    New-Route 'POST' '/api/v1/atenea/reportes/enviar-whatsapp' '/api/v1/reportes/imagen'
    New-Route 'POST' '/api/v1/atenea/whatsapp/enviar-media' '/api/v1/reportes/imagen'

    New-Route 'GET' '/api/admin/cola' '/api/v1/admin/cola'
    New-Route 'GET' '/api/v1/atenea/admin/cola' '/api/v1/admin/cola'
    New-Route 'PUT' '/api/admin/cola/:hashLargo' '/api/v1/admin/cola/:hashLargo'
    New-Route 'DELETE' '/api/admin/cola/:hashLargo' '/api/v1/admin/cola/:hashLargo'
)

$Removed = @(
    New-Route 'POST' '/api/tasas/n8n-webhook' '/api/v1/tasas/guardar-lote'
)

$Expected = @{}
foreach ($route in @($Canonical + $Aliases + $Removed)) {
    $Expected["$($route.Method) $($route.Path)"] = $route
}

Push-Location $Root
try {
    $env:GLAUK_ROUTE_MANIFEST = '1'
    $output = & go test ./cmd/server -run '^TestRouteManifest$' -count=1 -v 2>&1
    $exitCode = $LASTEXITCODE
    if ($exitCode -ne 0) {
        $output | ForEach-Object { Write-Host $_ }
        throw "no se pudo ejecutar TestRouteManifest (exit $exitCode)"
    }
}
finally {
    Remove-Item Env:GLAUK_ROUTE_MANIFEST -ErrorAction SilentlyContinue
    Pop-Location
}

$actual = @()
foreach ($line in $output) {
    if ([string]$line -match 'ROUTE\|([A-Z]+)\|(\S+)') {
        $actual += New-Route $Matches[1] $Matches[2]
    }
}
if ($actual.Count -eq 0) {
    throw 'TestRouteManifest no imprimió rutas'
}

$surface = @($actual | Where-Object {
    $_.Path -like '/api/*' -or $_.Path -eq '/health' -or $_.Path -eq '/ready'
})
$actualKeys = @{}
$duplicates = @()
foreach ($route in $surface) {
    $key = "$($route.Method) $($route.Path)"
    if ($actualKeys.ContainsKey($key)) {
        $duplicates += $key
    }
    $actualKeys[$key] = $true
}

$failures = [System.Collections.Generic.List[string]]::new()
foreach ($route in $Canonical) {
    $key = "$($route.Method) $($route.Path)"
    if (-not $actualKeys.ContainsKey($key)) {
        $failures.Add("falta la ruta canónica: $key")
    }
}
foreach ($route in $Aliases) {
    $key = "$($route.Method) $($route.Path)"
    if (-not $actualKeys.ContainsKey($key)) {
        $failures.Add("falta el alias histórico: $key -> $($route.Successor)")
    }
}
foreach ($route in $Removed) {
    $key = "$($route.Method) $($route.Path)"
    if (-not $actualKeys.ContainsKey($key)) {
        $failures.Add("falta la ruta retirada explícita: $key")
    }
}
foreach ($route in $surface) {
    $key = "$($route.Method) $($route.Path)"
    if (-not $Expected.ContainsKey($key)) {
        $failures.Add("ruta registrada fuera de la matriz §5: $key")
    }
}
foreach ($key in $duplicates) {
    $failures.Add("ruta duplicada Method + Path: $key")
}

$canonicalCount = @($Canonical | Where-Object {
    $actualKeys.ContainsKey("$($_.Method) $($_.Path)")
}).Count
Write-Host ("Canónicas: {0}/{1}" -f $canonicalCount, $Canonical.Count)
Write-Host ("Alias: {0}/{1}" -f @($Aliases | Where-Object { $actualKeys.ContainsKey("$($_.Method) $($_.Path)") }).Count, $Aliases.Count)
Write-Host ("Rutas retiradas explícitas: {0}/{1}" -f @($Removed | Where-Object { $actualKeys.ContainsKey("$($_.Method) $($_.Path)") }).Count, $Removed.Count)
Write-Host ("Superficie API/sondas: {0} rutas" -f $surface.Count)

if ($failures.Count -gt 0) {
    foreach ($failure in $failures) {
        Write-Host "ERROR: $failure" -ForegroundColor Red
    }
    exit 1
}

Write-Host 'OK: superficie HTTP congelada, cero colisiones y matriz §5 completa.' -ForegroundColor Green
exit 0
