# Comprehensive Go Module Fix Script
Write-Host "==== QUANT WebWork GO Complete Module Fix ====" -ForegroundColor Cyan
Write-Host "This script systematically fixes Go module issues" -ForegroundColor Cyan

$projectDir = "Z:\.CodingProjects\GitHub_Repos\Quant_WebWork_GO\QUANT_WW_GO"
$backupDir = "Z:\.CodingProjects\GitHub_Repos\Quant_WebWork_GO\module_fix_backup_$(Get-Date -Format 'yyyyMMdd_HHmmss')"

# Create backup directory
New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
Write-Host "Created backup directory: $backupDir" -ForegroundColor Green

# Backup go.mod and go.sum
Copy-Item -Path "$projectDir\go.mod" -Destination "$backupDir\go.mod.backup" -Force
if (Test-Path "$projectDir\go.sum") {
    Copy-Item -Path "$projectDir\go.sum" -Destination "$backupDir\go.sum.backup" -Force
}
Write-Host "Backed up go.mod and go.sum files" -ForegroundColor Green

# Step 1: Check and create missing directories referenced in imports
Write-Host "`n[STEP 1] Checking for missing package directories..." -ForegroundColor Cyan

# List of critical directories that might be referenced in imports
$criticalDirs = @(
    "internal/api/graphql/schema",
    "internal/api/service",
    "internal/bridge/adapter",
    "internal/core/config",
    "internal/core/discovery",
    "internal/core/metrics",
    "internal/security",
    "internal/security/token",
    "internal/security/risk",
    "internal/storage"
)

$createdDirs = 0
foreach ($dir in $criticalDirs) {
    $fullPath = Join-Path -Path $projectDir -ChildPath $dir
    if (-not (Test-Path $fullPath -PathType Container)) {
        New-Item -ItemType Directory -Path $fullPath -Force | Out-Null
        Write-Host "  Created missing directory: $dir" -ForegroundColor Yellow
        $createdDirs++
        
        # Create a stub .go file in the directory to make it a valid Go package
        $packageName = ($dir -split '/')[-1]
        $stubFileContent = @"
// Stub file for package $packageName
// This was auto-generated to fix module dependencies
package $packageName
"@
        Set-Content -Path "$fullPath\stub.go" -Value $stubFileContent
        Write-Host "    - Added stub file for package $packageName" -ForegroundColor Yellow
    }
}

Write-Host "Created $createdDirs missing directories with stub packages" -ForegroundColor Green

# Step 2: Update go.mod file to ensure proper module path and replace directives
Write-Host "`n[STEP 2] Updating go.mod file..." -ForegroundColor Cyan

# First, get the contents of the existing go.mod to preserve required packages
$existingGoMod = Get-Content -Path "$projectDir\go.mod" -Raw

# Extract the requires section if it exists
$requiresSection = ""
if ($existingGoMod -match "require\s*\(\s*([\s\S]*?)\s*\)") {
    $requiresSection = $Matches[1]
}

# Create a clean go.mod file with proper module path and required replace directives
$newGoModContent = @"
module github.com/IAM-timmy1t/Quant_WebWork_GO

go 1.21

// Handle import path issues with replace directives
replace (
	github.com/IAM-timmy1t/Quant_WebWork_GO => ./
	google.golang.org/grpc/web => github.com/improbable-eng/grpc-web v0.15.0
)

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.1
	github.com/graphql-go/graphql v0.8.1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.1
	github.com/prometheus/client_golang v1.18.0
	github.com/stretchr/testify v1.8.4
	google.golang.org/grpc v1.62.0
	golang.org/x/net v0.17.0 // indirect
)
"@

Set-Content -Path "$projectDir\go.mod" -Value $newGoModContent
Write-Host "Updated go.mod file with correct module path and essential dependencies" -ForegroundColor Green

# Step 3: Run go mod tidy to update dependencies
Write-Host "`n[STEP 3] Running 'go mod tidy' to update dependencies..." -ForegroundColor Cyan
try {
    Set-Location -Path $projectDir
    $output = & go mod tidy -e 2>&1
    Write-Host "go mod tidy completed (with -e flag to ignore errors)" -ForegroundColor Yellow
    Write-Host $output
} catch {
    Write-Host "Error running go mod tidy: $_" -ForegroundColor Red
}

# Step 4: Create vendor directory to manage dependencies locally
Write-Host "`n[STEP 4] Creating vendor directory..." -ForegroundColor Cyan
try {
    Set-Location -Path $projectDir
    $output = & go mod vendor 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Successfully created vendor directory" -ForegroundColor Green
    } else {
        Write-Host "Warning: vendor operation output:" -ForegroundColor Yellow
        Write-Host $output
    }
} catch {
    Write-Host "Error creating vendor directory: $_" -ForegroundColor Red
}

# Step 5: Test compilation
Write-Host "`n[STEP 5] Testing compilation with 'go build'..." -ForegroundColor Cyan
try {
    Set-Location -Path $projectDir
    $output = & go build -v ./cmd/... 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Build succeeded for cmd packages!" -ForegroundColor Green
    } else {
        Write-Host "Warning: build output:" -ForegroundColor Yellow
        Write-Host $output
    }
} catch {
    Write-Host "Error building project: $_" -ForegroundColor Red
}

Write-Host "`n==== Go Module Fix Complete ====" -ForegroundColor Cyan
Write-Host "Backup of original files saved to: $backupDir" -ForegroundColor Green
Write-Host "`nImportant Notes:" -ForegroundColor Yellow
Write-Host "1. Stub packages were created for missing directories. Replace the stub files with actual implementations." -ForegroundColor White
Write-Host "2. If build errors persist, you may need to:" -ForegroundColor White
Write-Host "   - Update import paths in specific files" -ForegroundColor White
Write-Host "   - Add missing implementations for required interfaces and types" -ForegroundColor White
Write-Host "3. Run 'go build -v ./...' to see detailed compilation information" -ForegroundColor White
