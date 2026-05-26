# Aura TUI Windows Installation Script
# Auto-detects processor architecture and registers Aura globally in your system path

$ErrorActionPreference = "Stop"

# Detect System Architecture
$Arch = $env:PROCESSOR_ARCHITECTURE
$BinaryName = ""

if ($Arch -eq "AMD64") {
    $BinaryName = "aura-windows-amd64.exe"
    Write-Host "Detected Architecture: 64-bit Intel/AMD (x64)" -ForegroundColor Cyan
} elseif ($Arch -eq "ARM64") {
    $BinaryName = "aura-windows-arm64.exe"
    Write-Host "Detected Architecture: 64-bit ARM (arm64)" -ForegroundColor Cyan
} else {
    $BinaryName = "aura-windows-386.exe"
    Write-Host "Detected Architecture: 32-bit Legacy (x86)" -ForegroundColor Cyan
}

$InstallDir = "$env:USERPROFILE\AppData\Local\aura"
$TargetPath = "$InstallDir\aura.exe"

# Create installation directory if missing
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir | Out-Null
}

# If running locally from a build folder, we can copy aura.exe directly if it exists
if (Test-Path "aura.exe") {
    Write-Host "Copying locally compiled binary to installation folder..." -ForegroundColor Cyan
    Copy-Item -Path "aura.exe" -Destination $TargetPath -Force
} else {
    # Download matching pre-compiled release from GitHub
    $ReleaseUrl = "https://github.com/Sussysham/aura-go/releases/latest/download/$BinaryName"
    Write-Host "Downloading $BinaryName from GitHub..." -ForegroundColor Cyan
    
    try {
        Invoke-WebRequest -Uri $ReleaseUrl -OutFile $TargetPath -UseBasicParsing
    } catch {
        Write-Error "Failed to download pre-compiled release! If the repository is private or not yet published, compile natively by running: go build -o aura.exe main.go"
        exit 1
    }
}

# Register User PATH
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    Write-Host "Registering Aura in your User environment PATH..." -ForegroundColor Cyan
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "Aura TUI registered successfully!" -ForegroundColor Green
} else {
    Write-Host "Aura TUI is already registered in your PATH!" -ForegroundColor Green
}

Write-Host "Aura TUI Installation Complete!" -ForegroundColor Green
Write-Host "--------------------------------------------------------" -ForegroundColor DarkGray
Write-Host "Please RE-OPEN your terminal and run 'aura' from anywhere!" -ForegroundColor Yellow
Write-Host "--------------------------------------------------------" -ForegroundColor DarkGray
