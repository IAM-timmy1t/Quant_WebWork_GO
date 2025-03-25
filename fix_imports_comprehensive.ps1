# Comprehensive script to fix Go module import errors
Write-Host "==== QUANT WebWork GO Import Fixer ====" -ForegroundColor Cyan
Write-Host "This script will fix import paths and module configuration issues" -ForegroundColor Cyan
Write-Host "Running a comprehensive fix for Go module import errors..." -ForegroundColor Yellow

$projectDir = "Z:\.CodingProjects\GitHub_Repos\Quant_WebWork_GO\QUANT_WW_GO"
$backupDir = "Z:\.CodingProjects\GitHub_Repos\Quant_WebWork_GO\backup_$(Get-Date -Format 'yyyyMMdd_HHmmss')"

# Create backup directory
New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
Write-Host "Created backup directory: $backupDir" -ForegroundColor Green

# Backup go.mod and go.sum
Copy-Item -Path "$projectDir\go.mod" -Destination "$backupDir\go.mod.backup" -Force
Copy-Item -Path "$projectDir\go.sum" -Destination "$backupDir\go.sum.backup" -Force
Write-Host "Backed up go.mod and go.sum files" -ForegroundColor Green

# 1. Fix imports in Go files
Write-Host "`n[STEP 1] Fixing import paths in Go files..." -ForegroundColor Cyan
$goFiles = Get-ChildItem -Path $projectDir -Filter "*.go" -Recurse
Write-Host "Found $($goFiles.Count) Go files to check for imports" -ForegroundColor White

$changedFiles = 0
$importPatterns = @(
    @{
        Pattern = "github.com/quant-webworks/go/"
        Replacement = "github.com/IAM-timmy1t/Quant_WebWork_GO/"
        Description = "Replaced old repository reference"
    },
    @{
        Pattern = "github.com/IAM-timmy1t/Quant_WebWork_GO/QUANT_WW_GO/"
        Replacement = "github.com/IAM-timmy1t/Quant_WebWork_GO/"
        Description = "Fixed redundant path segment"
    }
)

foreach ($file in $goFiles) {
    $content = Get-Content -Path $file.FullName -Raw
    $originalContent = $content
    $fileChanged = $false
    
    foreach ($pattern in $importPatterns) {
        if ($content -match $pattern.Pattern) {
            $content = $content -replace $pattern.Pattern, $pattern.Replacement
            Write-Host "  - $($file.Name): $($pattern.Description)" -ForegroundColor Yellow
            $fileChanged = $true
        }
    }
    
    if ($fileChanged) {
        Set-Content -Path $file.FullName -Value $content
        $changedFiles++
    }
}

Write-Host "Total files with fixed imports: $changedFiles" -ForegroundColor Green

# 2. Fix go.mod file
Write-Host "`n[STEP 2] Updating go.mod file..." -ForegroundColor Cyan
$goModPath = Join-Path $projectDir "go.mod"

if (Test-Path $goModPath) {
    # Create a fixed go.mod with correct module path and replace directives
    $newGoModContent = @"
module github.com/IAM-timmy1t/Quant_WebWork_GO

go 1.21

// Handle import path issues with replace directives
replace (
	github.com/quant-webworks/go => ./
	github.com/IAM-timmy1t/Quant_WebWork_GO/QUANT_WW_GO => ./
	google.golang.org/grpc/web => github.com/improbable-eng/grpc-web v0.15.0
)
"@

    Set-Content -Path $goModPath -Value $newGoModContent
    Write-Host "Updated go.mod with correct module path and replace directives" -ForegroundColor Green
    
    # Now run go mod tidy to rebuild dependency list
    Write-Host "`n[STEP 3] Running 'go mod tidy' to update dependencies..." -ForegroundColor Cyan
    
    # Change to the project directory and run go mod tidy
    Push-Location $projectDir
    try {
        $env:GO111MODULE = "on"
        $output = & go mod tidy 2>&1
        $success = $LASTEXITCODE -eq 0
        
        if ($success) {
            Write-Host "Successfully updated Go dependencies" -ForegroundColor Green
        } else {
            Write-Host "Error running 'go mod tidy':" -ForegroundColor Red
            Write-Host $output -ForegroundColor Red
            
            # Restore from backup if failed
            Write-Host "Restoring go.mod and go.sum from backup..." -ForegroundColor Yellow
            Copy-Item -Path "$backupDir\go.mod.backup" -Destination "$projectDir\go.mod" -Force
            Copy-Item -Path "$backupDir\go.sum.backup" -Destination "$projectDir\go.sum" -Force
        }
    } finally {
        Pop-Location
    }
} else {
    Write-Host "go.mod not found at $goModPath" -ForegroundColor Red
}

# 4. Verify the module
Write-Host "`n[STEP 4] Verifying Go module configuration..." -ForegroundColor Cyan
Push-Location $projectDir
try {
    $verifyOutput = & go mod verify 2>&1
    $verifySuccess = $LASTEXITCODE -eq 0
    
    if ($verifySuccess) {
        Write-Host "Module verification successful!" -ForegroundColor Green
    } else {
        Write-Host "Module verification found issues:" -ForegroundColor Yellow
        Write-Host $verifyOutput
    }
    
    # Show the current module dependencies
    Write-Host "`nCurrent module dependencies:" -ForegroundColor Cyan
    & go list -m all
} finally {
    Pop-Location
}

Write-Host "`n==== Import Fix Process Complete ====" -ForegroundColor Cyan
Write-Host "Backup of original files saved to: $backupDir" -ForegroundColor Cyan
Write-Host "Manual steps that may be needed:" -ForegroundColor Yellow
Write-Host "1. Check any remaining import errors in your IDE" -ForegroundColor Yellow
Write-Host "2. You may need to run 'go get' for specific packages" -ForegroundColor Yellow
Write-Host "3. If issues persist, you might need to modify the replace directives in go.mod" -ForegroundColor Yellow
