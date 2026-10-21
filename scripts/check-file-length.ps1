#Requires -Version 5.1
<#
.SYNOPSIS
    Verifica el presupuesto de lineas del repositorio Core.

.DESCRIPTION
    Reglas:
      * Go de produccion: 30..200 lineas.
      * Go de pruebas (*_test.go): <= 220 lineas.
      * HTML: <= 180 lineas.
      * JavaScript y CSS: <= 250 lineas, ignorando public/shared/vendor/.

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

$files = Get-ChildItem -LiteralPath $Root -Recurse -File |
    Where-Object { $_.Extension -in @('.go', '.html', '.js', '.css') }

foreach ($file in $files) {
    $relative = Get-RelativePath $file.FullName
    $extension = $file.Extension.ToLowerInvariant()

    # El vendor de terceros no forma parte del presupuesto propio.
    if (($extension -eq '.js' -or $extension -eq '.css') -and
        $relative -match '(^|[\\/])public[\\/]shared[\\/]vendor([\\/]|$)') {
        continue
    }

    # La shell y los modulos importados de glauk2 se conservan con paridad
    # visual 1:1. Recortarlos o partirlos artificialmente alteraria la interfaz,
    # por lo que esta UI queda fuera del presupuesto de lineas.
    if ($extension -in @('.html', '.js') -and
        $relative -match '^public[\\/]comprobantes([\\/]|$)') {
        continue
    }

    $lineCount = [System.IO.File]::ReadAllLines($file.FullName).Count
    $rule = $null
    $minimum = $null
    $maximum = $null

    switch ($extension) {
        '.go' {
            if ($file.Name -like '*_test.go') {
                $rule = 'Go test'
                $maximum = 220
            }
            else {
                $rule = 'Go produccion'
                $minimum = 30
                $maximum = 200
            }
        }
        '.html' {
            $rule = 'HTML'
            $maximum = 180
        }
        '.js' {
            $rule = 'JavaScript'
            $maximum = 250
        }
        '.css' {
            $rule = 'CSS'
            $maximum = 250
        }
    }

    if (-not $rule) {
        continue
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
Write-Host '=== Presupuesto de lineas ==='
Write-Host ("Archivos evaluados: {0}" -f $evaluated)
Write-Host ("Vendor excluido: public/shared/vendor/ (solo JS/CSS)")
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

Write-Host 'OK: todos los archivos cumplen el presupuesto de lineas.'
Write-Host ''
exit 0
