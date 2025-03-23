# scripts/setup_and_run.ps1
Write-Host "🔧 Starting QUANT_WW_GO full setup..." -ForegroundColor Cyan

# Ensure script runs from project root
Set-Location "$PSScriptRoot\.."

Write-Host "📦 Installing Go modules..." -ForegroundColor Green
go mod tidy
go mod verify

Write-Host "🎨 Installing React frontend dependencies..." -ForegroundColor Green
Set-Location "web\client"
npm install

Write-Host "⬅ Returning to project root..." -ForegroundColor Yellow
Set-Location "../.."

Write-Host "🐳 Starting Docker stack (backend + frontend + monitoring)..." -ForegroundColor Magenta
docker-compose up --build -d

Write-Host ""
Write-Host "✅ DONE! Your services are running:" -ForegroundColor Cyan
Write-Host "  🌐 Backend     → http://localhost:8080"
Write-Host "  🎨 Frontend    → http://localhost:8080"
Write-Host "  📈 Prometheus  → http://localhost:9090"
Write-Host "  📊 Grafana     → http://localhost:3000 (admin/admin)"
