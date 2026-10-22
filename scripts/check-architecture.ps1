#Requires -Version 5.1
<#
.SYNOPSIS
    Verifica las prohibiciones arquitectonicas del backend Go.

.DESCRIPTION
    El frontend vive en el repositorio glauk-ui; este script solo inspecciona
    codigo Go de produccion del backend. Comprueba:
      1. internal/reportes (raiz y http) no depende de pgx, perfiles,
         comprobantes, tasas ni platform/postgres.
      2. Solo platform/config lee variables de entorno.
      3. Solo cmd/server y platform/postgres construyen pgxpool.
      4. Solo internal/perfiles escribe en perfiles_glaukov.
      5. internal/platform/* no importa ningun dominio interno.
      6. internal/reportes (raiz y http) no contiene SQL.

    Sale con 1 si alguna comprobacion falla.
#>
[CmdletBinding()]
param(
    [string]$Root
)

$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($Root)) {
    $Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
}
$Root = (Resolve-Path -LiteralPath $Root).Path
$failures = New-Object System.Collections.Generic.List[string]

function Add-Failure {
    param([string]$Message)
    $script:failures.Add($Message)
}

function Get-RelativePath {
    param([string]$Path)
    return $Path.Substring($Root.Length).TrimStart('\', '/')
}

function Get-ProductionGoFiles {
    Get-ChildItem -LiteralPath $Root -Recurse -File -Filter '*.go' |
        Where-Object { $_.Name -notlike '*_test.go' }
}

function Get-GoListLine {
    param([string[]]$Packages)
    Push-Location -LiteralPath $Root
    try {
        $output = & go list -f '{{.ImportPath}};{{range .Imports}}{{.}};{{end}}' @Packages 2>&1
        $exitCode = $LASTEXITCODE
    }
    finally {
        Pop-Location
    }
    if ($exitCode -ne 0) {
        throw "go list fallo: $output"
    }
    return $output
}

function Test-ForbiddenImports {
    param(
        [string[]]$Packages,
        [string]$Description,
        [string]$ForbiddenPattern
    )
    $lines = Get-GoListLine -Packages $Packages
    foreach ($line in $lines) {
        if (-not $line -or $line -notmatch ';') {
            continue
        }
        $parts = $line -split ';'
        $package = $parts[0]
        foreach ($import in $parts[1..($parts.Count - 1)]) {
            if (-not $import) {
                continue
            }
            if ($import -match $ForbiddenPattern) {
                Add-Failure "${Description}: $package importa $import"
            }
        }
    }
}

Write-Host ''
Write-Host '=== Arquitectura Glauk ==='
Write-Host ''

# 1) Aislamiento del dominio reportes (raiz y capa HTTP).
try {
    $reportesForbidden = 'github\.com/jackc/pgx|glauk/internal/(perfiles|comprobantes|tasas)(/|$)|glauk/internal/platform/(postgres|evolution)(/|$)'
    Test-ForbiddenImports -Packages @('./internal/reportes', './internal/reportes/http') `
        -Description 'reportes' -ForbiddenPattern $reportesForbidden

    Push-Location -LiteralPath $Root
    try {
        $transitive = & go list -deps ./internal/reportes ./internal/reportes/http 2>&1
        $transitiveExit = $LASTEXITCODE
    }
    finally {
        Pop-Location
    }
    if ($transitiveExit -ne 0) {
        throw "go list -deps fallo: $transitive"
    }
    foreach ($dependency in $transitive) {
        if ($dependency -match 'github\.com/jackc/pgx|glauk/internal/(perfiles|comprobantes|tasas)(/|$)|glauk/internal/platform/(postgres|evolution)(/|$)') {
            Add-Failure "reportes: dependencia transitiva prohibida $dependency"
        }
    }
}
catch {
    Add-Failure "reportes: no se pudo verificar imports ($($_.Exception.Message))"
}
Write-Host '[1/6] reportes sin pgx, dominios ni platform/postgres: verificado'

# 2) Lectura de entorno solo en platform/config.
$envPattern = '\bos\.(Getenv|LookupEnv)\s*\(|\bgodotenv\b'
foreach ($file in Get-ProductionGoFiles) {
    $relative = Get-RelativePath $file.FullName
    if ($relative -match '(^|[\\/])platform[\\/]config([\\/]|$)') {
        continue
    }
    $matches = Select-String -LiteralPath $file.FullName -Pattern $envPattern
    foreach ($match in $matches) {
        Add-Failure "entorno: ${relative}:$($match.LineNumber) usa $($match.Line.Trim())"
    }
}
Write-Host '[2/6] entorno solo en platform/config: verificado'

# 3) Construccion de pgxpool solo en los bordes de infraestructura.
$poolPattern = '\bpgxpool\.New(WithConfig)?\s*\('
foreach ($file in Get-ProductionGoFiles) {
    $relative = Get-RelativePath $file.FullName
    if ($relative -match '(^|[\\/])cmd[\\/]server([\\/]|$)' -or
        $relative -match '(^|[\\/])platform[\\/]postgres([\\/]|$)') {
        continue
    }
    $matches = Select-String -LiteralPath $file.FullName -Pattern $poolPattern
    foreach ($match in $matches) {
        Add-Failure "pgxpool: ${relative}:$($match.LineNumber) construye el pool fuera de cmd/server o platform/postgres"
    }
}
Write-Host '[3/6] pgxpool solo en cmd/server o platform/postgres: verificado'

# 4) Escrituras de perfiles_glaukov solo en internal/perfiles.
$perfilesWritePattern = '(?i)\b(INSERT\s+INTO|UPDATE|DELETE\s+FROM)\s+["'']?perfiles_glaukov\b'
foreach ($file in Get-ProductionGoFiles) {
    $relative = Get-RelativePath $file.FullName
    if ($relative -match '(^|[\\/])internal[\\/]perfiles([\\/]|$)') {
        continue
    }
    $matches = Select-String -LiteralPath $file.FullName -Pattern $perfilesWritePattern
    foreach ($match in $matches) {
        Add-Failure "perfiles_glaukov: ${relative}:$($match.LineNumber) escribe fuera de internal/perfiles"
    }
}
Write-Host '[4/6] escrituras de perfiles_glaukov solo en internal/perfiles: verificado'

# 5) La infraestructura no depende de los dominios.
try {
    $platformLines = Get-GoListLine -Packages @('./internal/platform/...')
    foreach ($line in $platformLines) {
        if (-not $line -or $line -notmatch ';') {
            continue
        }
        $parts = $line -split ';'
        $package = $parts[0]
        foreach ($import in $parts[1..($parts.Count - 1)]) {
            if ($import -match 'glauk/internal/(tasas|perfiles|comprobantes|reportes)(/|$)') {
                Add-Failure "platform: $package importa el dominio $import"
            }
        }
    }
}
catch {
    Add-Failure "platform: no se pudo verificar imports ($($_.Exception.Message))"
}
Write-Host '[5/6] platform sin imports de dominios: verificado'

# 6) El dominio reportes no contiene SQL (los adaptadores quedan excluidos).
$sqlPattern = '(?i)\b(SELECT\s+[^\r\n]+\s+FROM|INSERT\s+INTO|UPDATE\s+\w+\s+SET|DELETE\s+FROM)\b'
$reportesRoot = Join-Path $Root 'internal/reportes'
if (Test-Path -LiteralPath $reportesRoot) {
    foreach ($file in Get-ChildItem -LiteralPath $reportesRoot -Recurse -File -Filter '*.go') {
        if ($file.Name -like '*_test.go') {
            continue
        }
        $relative = Get-RelativePath $file.FullName
        if ($relative -match '(^|[\\/])reportes[\\/]adapters([\\/]|$)') {
            continue
        }
        $matches = Select-String -LiteralPath $file.FullName -Pattern $sqlPattern
        foreach ($match in $matches) {
            Add-Failure "SQL: ${relative}:$($match.LineNumber) contiene SQL en el dominio reportes"
        }
    }
}
Write-Host '[6/6] reportes sin SQL: verificado'
Write-Host ''

if ($failures.Count -gt 0) {
    Write-Host ("VIOLACIONES DE ARQUITECTURA: {0}" -f $failures.Count)
    Write-Host ''
    foreach ($failure in $failures) {
        Write-Host "  - $failure"
    }
    Write-Host ''
    exit 1
}

Write-Host 'OK: no se detectaron violaciones de arquitectura.'
Write-Host ''
exit 0
