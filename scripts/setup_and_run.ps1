# Setup and run script for QUANT_WW_GO
# Simple version with minimal dependencies

Write-Host "Starting QUANT_WW_GO setup..." -ForegroundColor Cyan

# Navigate to project directory
$projectDir = Join-Path $PSScriptRoot "..\QUANT_WW_GO"
Write-Host "Setting project directory to: $projectDir" -ForegroundColor Yellow
Set-Location -Path $projectDir

# Check if go.mod exists
if (Test-Path "go.mod") {
    Write-Host "Installing Go modules..." -ForegroundColor Green
    go mod tidy
    go mod verify
} else {
    Write-Host "Warning: go.mod not found in $projectDir" -ForegroundColor Yellow
}

# Check for frontend
$frontendPath = Join-Path $projectDir "web\client"
if (Test-Path $frontendPath) {
    Write-Host "Installing frontend dependencies..." -ForegroundColor Green
    Set-Location $frontendPath
    npm install
    Set-Location $projectDir
} else {
    Write-Host "Warning: Frontend directory not found at $frontendPath" -ForegroundColor Yellow
}

# Check for docker-compose
if (Test-Path "docker-compose.yml") {
    Write-Host "Starting Docker services..." -ForegroundColor Magenta
    docker-compose up --build -d
    
    Write-Host "Services running at:" -ForegroundColor Cyan
    Write-Host "  - Backend     -> http://localhost:8080" -ForegroundColor Green
    Write-Host "  - Frontend    -> http://localhost:8080" -ForegroundColor Green
    Write-Host "  - Prometheus  -> http://localhost:9090" -ForegroundColor Green
    Write-Host "  - Grafana     -> http://localhost:3000 (admin/admin)" -ForegroundColor Green
} else {
    Write-Host "Warning: docker-compose.yml not found in $projectDir" -ForegroundColor Yellow
}

Write-Host "Setup complete!" -ForegroundColor Cyan
