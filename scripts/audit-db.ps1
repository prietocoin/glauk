[CmdletBinding()]
param(
    [string]$Dsn,
    [string]$SqlFile,
    [string]$CsvPath,
    [switch]$Simulate,
    [switch]$Detailed
)

$ErrorActionPreference = 'Stop'
$script:report = [System.Collections.Generic.List[object]]::new()
if ([string]::IsNullOrWhiteSpace($SqlFile)) {
    $baseDir = if ($PSScriptRoot) { $PSScriptRoot } else { Split-Path -Parent $PSCommandPath }
    $SqlFile = Join-Path $baseDir 'audit-pre-cutover.sql'
}

# Extrae los bloques del SQL y valida que solo contengan SELECT/WITH.
function Get-AuditBlocks {
    param([string]$Path)

    if (-not (Test-Path -LiteralPath $Path)) {
        throw "No existe el SQL de auditoria: $Path"
    }
    $sql = Get-Content -LiteralPath $Path -Raw -Encoding UTF8
    $blockMatches = [regex]::Matches($sql, '(?m)^-- BLOQUE\s+(?<id>\d+)\s*\|\s*(?<title>[^\r\n]+)\r?$')
    if ($blockMatches.Count -eq 0) {
        throw "El SQL no contiene bloques '-- BLOQUE nn | titulo'."
    }

    $blocks = [System.Collections.Generic.List[object]]::new()
    for ($i = 0; $i -lt $blockMatches.Count; $i++) {
        $match = $blockMatches[$i]
        $start = $match.Index + $match.Length
        $end = if ($i + 1 -lt $blockMatches.Count) { $blockMatches[$i + 1].Index } else { $sql.Length }
        $chunk = $sql.Substring($start, $end - $start)
        $validText = [regex]::Match($chunk, '(?m)^-- VALIDA:\s*(?<value>.+)$').Groups['value'].Value.Trim()
        $actionText = [regex]::Match($chunk, '(?m)^-- ACCION:\s*(?<value>.+)$').Groups['value'].Value.Trim()
        if (-not $validText) { $validText = $match.Groups['title'].Value.Trim() }
        if (-not $actionText) { $actionText = 'Revisar hallazgos antes del cutover.' }

        $clean = [regex]::Replace($chunk, '(?m)^\s*--.*$', '')
        $clean = [regex]::Replace($clean, '(?is)\s*COMMIT\s*;?\s*$', '')
        $queryMatch = [regex]::Match($clean, '(?is)\b(SELECT|WITH)\b')
        if (-not $queryMatch.Success) {
            throw "El bloque $($match.Groups['id'].Value) no contiene SELECT/WITH."
        }
        $query = $clean.Substring($queryMatch.Index).Trim()
        $query = $query.TrimEnd().TrimEnd(';').Trim()
        if ($query -match '(?i)\b(INSERT|UPDATE|DELETE|ALTER|DROP|TRUNCATE|CREATE|GRANT|REVOKE|COPY|CALL|DO|VACUUM|ANALYZE|REFRESH|REINDEX|CLUSTER|SET|RESET)\b') {
            throw "El bloque $($match.Groups['id'].Value) contiene una operacion no permitida."
        }

        [void]$blocks.Add([pscustomobject]@{
            Id = $match.Groups['id'].Value
            Title = $match.Groups['title'].Value.Trim()
            Valida = $validText
            Accion = $actionText
            Query = $query
        })
    }
    return $blocks.ToArray()
}

# Ejecuta una consulta dentro de una transaccion READ ONLY y devuelve sus filas.
function Invoke-PsqlQuery {
    param(
        [string]$PsqlPath,
        [string]$Query,
        [string]$Dsn
    )

    $wrapped = "BEGIN TRANSACTION READ ONLY;`nSET LOCAL statement_timeout = '30s';`n$Query;`nCOMMIT;"
    $previousTimeout = $env:PGCONNECT_TIMEOUT
    try {
        $env:PGCONNECT_TIMEOUT = '10'
        $output = & $PsqlPath --no-psqlrc -v ON_ERROR_STOP=1 --tuples-only --csv --dbname $Dsn --command $wrapped 2>&1
        $code = $LASTEXITCODE
    } finally {
        if ($null -eq $previousTimeout) {
            Remove-Item Env:PGCONNECT_TIMEOUT -ErrorAction SilentlyContinue
        } else {
            $env:PGCONNECT_TIMEOUT = $previousTimeout
        }
    }
    if ($code -ne 0) {
        throw ($output -join [Environment]::NewLine)
    }
    return @($output | Where-Object { $_.Trim() -ne '' })
}

function Add-ReportRow {
    param($Block, [string]$Findings, [string]$State)
    [void]$script:report.Add([pscustomobject]@{
        Bloque = $Block.Id
        Valida = $Block.Valida
        Hallazgos = $Findings
        Estado = $State
        Accion = $Block.Accion
    })
}

if ([string]::IsNullOrWhiteSpace($Dsn)) {
    $Dsn = $env:DATABASE_URL
}
$hasDsn = -not [string]::IsNullOrWhiteSpace($Dsn)
$psql = Get-Command psql -ErrorAction SilentlyContinue
$usesSimulation = [bool]$Simulate
$incomplete = $false

if (-not $usesSimulation -and -not $psql) {
    Write-Warning 'psql no esta disponible; se ejecuta el parseo seguro en modo simulacion.'
    $usesSimulation = $true
    $incomplete = $true
}
if (-not $usesSimulation -and -not $hasDsn) {
    Write-Warning 'No hay DSN ni DATABASE_URL; se ejecuta el parseo seguro en modo simulacion.'
    $usesSimulation = $true
    $incomplete = $true
}

try {
    $blocks = Get-AuditBlocks -Path $SqlFile
} catch {
    Write-Host "ERROR: $($_.Exception.Message)" -ForegroundColor Red
    exit 3
}

Write-Host '=== Auditoria pre-cutover Glauk ===' -ForegroundColor Cyan
Write-Host "SQL: $SqlFile"
Write-Host "DSN: $(if ($hasDsn) { 'presente (oculto)' } else { 'ausente' })"
Write-Host "Modo: $(if ($usesSimulation) { 'simulacion / parseo seguro' } else { 'ejecucion real de solo lectura' })"

foreach ($block in $blocks) {
    if ($usesSimulation) {
        Add-ReportRow -Block $block -Findings 'n/d' -State 'SIMULADO'
        continue
    }
    try {
        $rows = Invoke-PsqlQuery -PsqlPath $psql.Source -Query $block.Query -Dsn $Dsn
        if ($rows.Count -eq 0) {
            Add-ReportRow -Block $block -Findings '0' -State 'OK'
        } else {
            Add-ReportRow -Block $block -Findings ([string]$rows.Count) -State 'REVISAR'
            if ($Detailed) {
                Write-Host "  $($block.Id) $($block.Title)" -ForegroundColor Yellow
                foreach ($row in $rows) { Write-Host "    $row" }
            }
        }
    } catch {
        Add-ReportRow -Block $block -Findings 'ERROR' -State 'ERROR'
        Write-Host "ERROR en bloque $($block.Id): $($_.Exception.Message)" -ForegroundColor Red
    }
}

$script:report | Format-Table -AutoSize | Out-String -Width 300 | Write-Host
if ($CsvPath) {
    $script:report | Export-Csv -LiteralPath $CsvPath -NoTypeInformation -Encoding UTF8
    Write-Host "Reporte CSV: $CsvPath"
}

$review = @($script:report | Where-Object { $_.Estado -eq 'REVISAR' }).Count
$errors = @($script:report | Where-Object { $_.Estado -eq 'ERROR' }).Count

if ($errors -gt 0) {
    Write-Host "Auditoria INCOMPLETA: $errors bloque(s) con error de ejecucion." -ForegroundColor Red
    exit 3
}
if ($incomplete) {
    Write-Host 'Auditoria NO EJECUTADA: falta psql o DSN. Use -Simulate para aceptar el parseo.' -ForegroundColor Yellow
    exit 2
}
if ($review -gt 0) {
    Write-Host "ACCION HUMANA REQUERIDA: $review bloque(s) con hallazgos antes de pasar trafico real." -ForegroundColor Red
    exit 1
}
if ($usesSimulation) {
    Write-Host 'SIMULACION OK: estructura y solo-lectura verificadas; no se consultaron datos.' -ForegroundColor Yellow
    exit 0
}
Write-Host 'AUDITORIA OK: no hay hallazgos pendientes en los bloques ejecutados.' -ForegroundColor Green
exit 0
