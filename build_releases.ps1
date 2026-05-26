# Aura TUI Cross-Architecture Release Builder
# Compiles binaries for standard 64-bit, ARM64, and legacy 32-bit Windows systems.

$ErrorActionPreference = "Stop"

# Create output folder
$DistDir = "dist"
if (-not (Test-Path $DistDir)) {
    New-Item -ItemType Directory -Path $DistDir | Out-Null
}

Write-Host "Starting Cross-Architecture Compilation Pipeline..." -ForegroundColor Cyan
Write-Host "--------------------------------------------------------" -ForegroundColor DarkGray

# 1. Windows x64 (AMD64)
Write-Host "Compiling for Windows x64 (64-bit standard)..." -ForegroundColor Yellow
$env:GOOS="windows"
$env:GOARCH="amd64"
go build -ldflags "-s -w" -o dist/aura-windows-amd64.exe main.go
Write-Host "   Generated: dist/aura-windows-amd64.exe" -ForegroundColor Green

# 2. Windows ARM64
Write-Host "Compiling for Windows ARM64 (Modern Snapdragon laptops)..." -ForegroundColor Yellow
$env:GOOS="windows"
$env:GOARCH="arm64"
go build -ldflags "-s -w" -o dist/aura-windows-arm64.exe main.go
Write-Host "   Generated: dist/aura-windows-arm64.exe" -ForegroundColor Green

# 3. Windows x86 (386)
Write-Host "Compiling for Windows x86 (32-bit legacy)..." -ForegroundColor Yellow
$env:GOOS="windows"
$env:GOARCH="386"
go build -ldflags "-s -w" -o dist/aura-windows-386.exe main.go
Write-Host "   Generated: dist/aura-windows-386.exe" -ForegroundColor Green

# Reset Go env vars
$env:GOOS=""
$env:GOARCH=""

Write-Host "All release binaries built successfully in ./dist/" -ForegroundColor Green
Write-Host "Ready for release publication!" -ForegroundColor Green
