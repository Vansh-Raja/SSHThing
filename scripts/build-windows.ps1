param(
    [string]$OutFile = "sshthing.exe"
)

$ErrorActionPreference = "Stop"

$mingwBin = "C:\msys64\mingw64\bin"
if (-not (Test-Path $mingwBin)) {
    throw "MSYS2 MinGW not found at $mingwBin. Install MSYS2 and mingw-w64 first."
}

$env:PATH = "$mingwBin;$env:PATH"
$env:CGO_ENABLED = "1"
$env:CC = "gcc"

# Suppress known sqlite/sqlcipher false-positive warning on some toolchains.
$existingCFlags = $env:CGO_CFLAGS
$suppressFlag = "-Wno-return-local-addr"
if ([string]::IsNullOrWhiteSpace($existingCFlags)) {
    $env:CGO_CFLAGS = $suppressFlag
} elseif ($existingCFlags -notmatch [regex]::Escape($suppressFlag)) {
    $env:CGO_CFLAGS = "$existingCFlags $suppressFlag"
}

Write-Host "CGO_ENABLED=$env:CGO_ENABLED"
Write-Host "CC=$env:CC"
Write-Host "CGO_CFLAGS=$env:CGO_CFLAGS"

go build -o $OutFile ./cmd/sshthing
Write-Host "Built $OutFile"
