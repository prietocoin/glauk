[CmdletBinding()]
param(
    [string]$BaseUrl = "http://localhost:3000",
    [int]$MaxHealthLatencyMs = 2000,
    [int]$TimeoutSec = 10
)

$ErrorActionPreference = "Stop"
$script:failures = [System.Collections.Generic.List[string]]::new()

function Add-Failure {
    param([string]$Message)
    $script:failures.Add($Message)
    Write-Host "FAIL  $Message" -ForegroundColor Red
}

function Convert-Headers {
    param($Headers)
    $map = @{}
    if ($null -eq $Headers) { return $map }
    if ($Headers -is [System.Collections.Specialized.NameValueCollection]) {
        foreach ($key in $Headers.AllKeys) { $map[$key] = [string]$Headers[$key] }
        return $map
    }
    foreach ($entry in $Headers.GetEnumerator()) {
        $map[$entry.Key] = ($entry.Value -join ", ")
    }
    return $map
}

function Get-HeaderValue {
    param($Headers, [string]$Name)
    foreach ($key in $Headers.Keys) {
        if ($key -ieq $Name) { return [string]$Headers[$key] }
    }
    return $null
}

function Invoke-Probe {
    param([string]$Name, [string]$Path)
    $uri = $BaseUrl + $Path
    $watch = [System.Diagnostics.Stopwatch]::StartNew()
    try {
        $response = Invoke-WebRequest -Uri $uri -Method Get -UseBasicParsing -TimeoutSec $TimeoutSec
    } catch {
        $watch.Stop()
        Add-Failure "$Name $Path no respondió: $($_.Exception.Message)"
        return $null
    }
    $watch.Stop()
    $headers = Convert-Headers $response.Headers
    return [pscustomobject]@{
        Name = $Name
        Path = $Path
        Status = [int]$response.StatusCode
        Headers = $headers
        Content = [string]$response.Content
        ElapsedMs = [math]::Round($watch.Elapsed.TotalMilliseconds, 0)
    }
}

function Convert-ProbeJson {
    param($Probe)
    try {
        return ($Probe.Content | ConvertFrom-Json)
    } catch {
        Add-Failure "$($Probe.Path) no devolvió JSON válido: $($_.Exception.Message)"
        return $null
    }
}

function Assert-SecurityHeaders {
    param($Probe)
    $expected = [ordered]@{
        "X-Content-Type-Options"       = "nosniff"
        "X-Frame-Options"              = "DENY"
        "Referrer-Policy"              = "no-referrer"
        "Permissions-Policy"           = "camera=(), microphone=(), geolocation=()"
        "Cross-Origin-Opener-Policy"   = "same-origin"
    }
    foreach ($name in $expected.Keys) {
        $actual = Get-HeaderValue $Probe.Headers $name
        if ($null -eq $actual) {
            Add-Failure "$($Probe.Path) no incluye la cabecera $name"
        } elseif ($actual -ne $expected[$name]) {
            Add-Failure "$($Probe.Path) $name = '$actual'; se esperaba '$($expected[$name])'"
        }
    }
    if ($BaseUrl -match "^https://") {
        $hsts = Get-HeaderValue $Probe.Headers "Strict-Transport-Security"
        if ($null -eq $hsts -or $hsts -notlike "max-age=31536000*") {
            Add-Failure "$($Probe.Path) no incluye HSTS válido"
        }
    }
}

if ([string]::IsNullOrWhiteSpace($BaseUrl)) { throw "BaseUrl es obligatorio" }
$BaseUrl = $BaseUrl.TrimEnd("/")
if ($MaxHealthLatencyMs -le 0) { throw "MaxHealthLatencyMs debe ser positivo" }
if ($TimeoutSec -le 0) { throw "TimeoutSec debe ser positivo" }

Write-Host "=== Smoke Glauk ===" -ForegroundColor Cyan
Write-Host "BaseUrl: $BaseUrl"

$health = Invoke-Probe "health" "/health"
$ready = Invoke-Probe "ready" "/ready"
$monitor = Invoke-Probe "monitor" "/api/v1/tasas/monitor"
$filters = Invoke-Probe "filtros" "/api/v1/reportes/filtros"

foreach ($probe in @($health, $ready, $monitor, $filters)) {
    if ($null -eq $probe) { continue }
    if ($probe.Status -ne 200) {
        Add-Failure "$($probe.Path) respondió $($probe.Status), se esperaba 200"
        continue
    }
    Write-Host ("OK    {0,-8} {1} {2} ms" -f $probe.Name, $probe.Status, $probe.ElapsedMs) -ForegroundColor Green
    Assert-SecurityHeaders $probe
}

if ($null -ne $health) {
    $body = Convert-ProbeJson $health
    if ($null -eq $body -or $body.status -ne "ok") {
        Add-Failure "/health no reporta status=ok"
    }
    if ($null -ne $body -and $body.service -ne "glauk") {
        Add-Failure "/health no reporta service=glauk"
    }
    if ($null -ne $body -and $body.checks.database.status -ne "ok") {
        Add-Failure "/health no reporta database.status=ok"
    }
    if ($health.ElapsedMs -gt $MaxHealthLatencyMs) {
        Add-Failure "/health tardó $($health.ElapsedMs) ms; umbral $MaxHealthLatencyMs ms"
    }
}

if ($null -ne $ready) {
    $body = Convert-ProbeJson $ready
    if ($null -eq $body -or $body.status -ne "ok") {
        Add-Failure "/ready no reporta status=ok"
    }
}

if ($null -ne $monitor) {
    $body = Convert-ProbeJson $monitor
    if ($null -eq $body -or $body.success -ne $true) {
        Add-Failure "/api/v1/tasas/monitor no reporta success=true"
    } elseif ($body.id_tasa -notmatch "^T\d+$") {
        Add-Failure "/api/v1/tasas/monitor no devuelve un id_tasa válido"
    } elseif ($null -eq $body.tasas) {
        Add-Failure "/api/v1/tasas/monitor no devuelve el objeto tasas"
    } elseif (@($body.tasas.PSObject.Properties).Count -eq 0) {
        Add-Failure "/api/v1/tasas/monitor devuelve un lote sin tasas"
    }
}

if ($null -ne $filters) {
    $body = Convert-ProbeJson $filters
    if ($null -eq $body -or $body.success -ne $true) {
        Add-Failure "/api/v1/reportes/filtros no reporta success=true"
    } elseif ($null -eq $body.data) {
        Add-Failure "/api/v1/reportes/filtros no devuelve data"
    } else {
        foreach ($field in @("roles", "entidades", "hashes")) {
            $property = $body.data.PSObject.Properties[$field]
            if ($null -eq $property -or $null -eq $property.Value) {
                Add-Failure "/api/v1/reportes/filtros no devuelve data.$field"
            } elseif ($property.Value -is [string] -or $property.Value -is [ValueType]) {
                Add-Failure "/api/v1/reportes/filtros data.$field no es una lista"
            }
        }
    }
}

if ($script:failures.Count -gt 0) {
    Write-Host "SMOKE FAIL: $($script:failures.Count) comprobación(es) fallida(s)." -ForegroundColor Red
    exit 1
}

Write-Host "SMOKE OK: servicio, datos y cabeceras de seguridad verificados." -ForegroundColor Green
exit 0
