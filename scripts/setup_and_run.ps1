# Setup and run script for Quant_WebWork_GO
# Following QUANT_AGI Implementation Protocol

Write-Host "[CSM-INFO] Starting Quant_WebWork_GO setup..." -ForegroundColor Cyan

# Navigate to project directory (root level)
$projectDir = Join-Path $PSScriptRoot ".."
Write-Host "[CSM-INFO] Setting project directory to: $projectDir" -ForegroundColor Yellow
Set-Location -Path $projectDir

# Backup important files before making changes
$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$backupDir = "backup_config_$timestamp"
New-Item -Path $backupDir -ItemType Directory -Force | Out-Null
Write-Host "[CSM-INFO] Created backup directory: $backupDir" -ForegroundColor Green

if (Test-Path "go.mod") {
    Copy-Item -Path "go.mod" -Destination "$backupDir/go.mod" -Force
    Copy-Item -Path "go.sum" -Destination "$backupDir/go.sum" -Force
    Write-Host "[CSM-INFO] Backed up go.mod and go.sum files" -ForegroundColor Green
}

# Set environment variable to bypass vendor directory
$env:GOFLAGS = "-mod=mod"
Write-Host "[CSM-INFO] Set GOFLAGS=-mod=mod to bypass vendor directory" -ForegroundColor Green

# Check if go.mod exists and handle module setup
if (Test-Path "go.mod") {
    Write-Host "[CSM-INFO] Installing Go modules..." -ForegroundColor Green
    
    # Run go mod tidy with error handling
    try {
        go mod tidy -e
        Write-Host "[CSM-INFO] Successfully ran go mod tidy" -ForegroundColor Green
    } catch {
        Write-Host "[CSM-ERR] Error during go mod tidy: $_" -ForegroundColor Red
    }
    
    # Verify modules
    try {
        go mod verify
        Write-Host "[CSM-INFO] Successfully verified modules" -ForegroundColor Green
    } catch {
        Write-Host "[CSM-ERR] Error during go mod verify: $_" -ForegroundColor Red
    }
} else {
    Write-Host "[CSM-WARN] Warning: go.mod not found in $projectDir" -ForegroundColor Yellow
}

# Check for frontend
$frontendPath = Join-Path $projectDir "web\client"
if (Test-Path $frontendPath) {
    Write-Host "[CSM-INFO] Installing frontend dependencies..." -ForegroundColor Green
    Set-Location $frontendPath
    npm install
    Set-Location $projectDir
} else {
    Write-Host "[CSM-WARN] Warning: Frontend directory not found at $frontendPath" -ForegroundColor Yellow
}

# Check for docker-compose
if (Test-Path "docker-compose.yml") {
    # Check if Docker is running first
    $dockerRunning = $false
    try {
        docker info > $null 2>&1
        $dockerRunning = $true
        Write-Host "[CSM-INFO] Docker is running" -ForegroundColor Green
    } catch {
        Write-Host "[CSM-WARN] Docker does not appear to be running. Please start Docker Desktop and try again." -ForegroundColor Yellow
    }
    
    if ($dockerRunning) {
        Write-Host "[CSM-INFO] Starting Docker services..." -ForegroundColor Magenta
        try {
            docker-compose up --build -d
            
            Write-Host "[CSM-INFO] Services running at:" -ForegroundColor Cyan
            Write-Host "  - Backend     -> http://localhost:8080" -ForegroundColor Green
            Write-Host "  - Frontend    -> http://localhost:8080" -ForegroundColor Green
            Write-Host "  - Prometheus  -> http://localhost:9090" -ForegroundColor Green
            Write-Host "  - Grafana     -> http://localhost:3000 (admin/admin)" -ForegroundColor Green
        } catch {
            Write-Host "[CSM-ERR] Error starting Docker services: $_" -ForegroundColor Red
        }
    }
} else {
    Write-Host "[CSM-WARN] Warning: docker-compose.yml not found in $projectDir" -ForegroundColor Yellow
}

# Build the application
Write-Host "[CSM-INFO] Building application..." -ForegroundColor Cyan
try {
    go build -v ./cmd/...
    Write-Host "[CSM-INFO] Build successful" -ForegroundColor Green
} catch {
    Write-Host "[CSM-ERR] Build failed: $_" -ForegroundColor Red
}

# Summary - QUANT_AGI Implementation Protocol
Write-Host "`n[CSM-INFO] Setup Summary:" -ForegroundColor Cyan
Write-Host "----------------------------------------" -ForegroundColor White
Write-Host "Project directory:    $projectDir" -ForegroundColor Green
Write-Host "Config backup in:     $backupDir" -ForegroundColor Green
Write-Host "Environment setup:    GOFLAGS=-mod=mod" -ForegroundColor Green
Write-Host "----------------------------------------" -ForegroundColor White

Write-Host "`n[CSM-INFO] Next steps:" -ForegroundColor Cyan
Write-Host "1. If changes made by this script didn't fully address Go module issues, run:" -ForegroundColor White
Write-Host "   . .\scripts\fix_test_imports.ps1" -ForegroundColor Yellow
Write-Host "2. For starting the application without Docker, run:" -ForegroundColor White
Write-Host "   go run ./cmd/server" -ForegroundColor Yellow

Write-Host "[CSM-INFO] Setup complete!" -ForegroundColor Cyan
