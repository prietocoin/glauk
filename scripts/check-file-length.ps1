#Requires -Version 5.1
<#
.SYNOPSIS
    Verifica el presupuesto de lineas del backend Go.

.DESCRIPTION
    Reglas:
      * Go de produccion: 30..200 lineas.
      * Go de pruebas (*_test.go): <= 220 lineas.

    El frontend vive en el repositorio glauk-ui; este script no evalua
    HTML, JavaScript ni CSS.

    Sale con 1 si existe al menos una violacion. Con el repositorio limpio
    imprime el resumen y sale con 0.
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

if (-not (Test-Path -LiteralPath $Root -PathType Container)) {
    Write-Host "ERROR: no existe el directorio raiz: $Root"
    exit 1
}

$violations = New-Object System.Collections.Generic.List[object]
$evaluated = 0

function Get-RelativePath {
    param([string]$Path)
    return $Path.Substring($Root.Length).TrimStart('\', '/')
}

$files = Get-ChildItem -LiteralPath $Root -Recurse -File -Filter '*.go'

foreach ($file in $files) {
    $relative = Get-RelativePath $file.FullName
    $lineCount = [System.IO.File]::ReadAllLines($file.FullName).Count

    if ($file.Name -like '*_test.go') {
        $rule = 'Go test'
        $minimum = $null
        $maximum = 220
    }
    else {
        $rule = 'Go produccion'
        $minimum = 30
        $maximum = 200
    }

    $evaluated++
    $reason = $null

    if ($null -ne $maximum -and $lineCount -gt $maximum) {
        $reason = "supera el maximo de $maximum lineas"
    }
    elseif ($null -ne $minimum -and $lineCount -lt $minimum) {
        $reason = "esta por debajo del minimo de $minimum lineas"
    }

    if ($reason) {
        $violations.Add([pscustomobject]@{
            Archivo = $relative
            Lineas  = $lineCount
            Regla   = $rule
            Motivo  = $reason
        })
    }
}

Write-Host ''
Write-Host '=== Presupuesto de lineas (Go) ==='
Write-Host ("Archivos evaluados: {0}" -f $evaluated)
Write-Host ''

if ($violations.Count -gt 0) {
    Write-Host ("VIOLACIONES ENCONTRADAS: {0}" -f $violations.Count)
    Write-Host ''
    foreach ($violation in $violations) {
        Write-Host ("  {0}: {1} lineas [{2}] - {3}" -f `
            $violation.Archivo, $violation.Lineas, $violation.Regla, $violation.Motivo)
    }
    Write-Host ''
    exit 1
}

Write-Host 'OK: todos los archivos Go cumplen el presupuesto de lineas.'
Write-Host ''
exit 0
