# Module Import Path Correction Script
Write-Host "==== QUANT WebWork GO Module Import Fixer ====" -ForegroundColor Cyan
Write-Host "This script will fix import paths to match the new module path" -ForegroundColor Cyan

$projectDir = "Z:\.CodingProjects\GitHub_Repos\Quant_WebWork_GO\QUANT_WW_GO"
$backupDir = "Z:\.CodingProjects\GitHub_Repos\Quant_WebWork_GO\backup_imports_$(Get-Date -Format 'yyyyMMdd_HHmmss')"

# Create backup directory
New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
Write-Host "Created backup directory: $backupDir" -ForegroundColor Green

# 1. Fix imports in Go files
Write-Host "`n[STEP 1] Fixing import paths in Go files..." -ForegroundColor Cyan
$goFiles = Get-ChildItem -Path $projectDir -Filter "*.go" -Recurse
Write-Host "Found $($goFiles.Count) Go files to check for imports" -ForegroundColor White

$changedFiles = 0
$oldImportPath = "github.com/IAM-timmy1t/Quant_WebWork_GO/"
$newImportPath = "github.com/IAM-timmy1t/Quant_WebWork_GO/QUANT_WW_GO/"

foreach ($file in $goFiles) {
    # Backup file
    Copy-Item -Path $file.FullName -Destination "$backupDir\$($file.Name).backup" -Force
    
    $content = Get-Content -Path $file.FullName -Raw
    $originalContent = $content
    $fileChanged = $false
    
    # Only replace the import path if it's not part of a comment and is an actual import statement
    if ($content -match "import\s+\([^)]*$oldImportPath" -or $content -match "import\s+""$oldImportPath") {
        $content = $content -replace "([^/])$oldImportPath", "`$1$newImportPath"
        Write-Host "  - Updated imports in $($file.FullName)" -ForegroundColor Yellow
        $fileChanged = $true
    }
    
    if ($fileChanged) {
        Set-Content -Path $file.FullName -Value $content
        $changedFiles++
    }
}

Write-Host "Total files with fixed imports: $changedFiles" -ForegroundColor Green

# After updating all imports, run go mod tidy
Write-Host "`n[STEP 2] Running go mod tidy to update dependencies..." -ForegroundColor Cyan
try {
    Set-Location -Path $projectDir
    $output = & go mod tidy 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "go mod tidy completed successfully!" -ForegroundColor Green
    } else {
        Write-Host "Error running 'go mod tidy':" -ForegroundColor Red
        Write-Host $output -ForegroundColor Red
    }
} catch {
    Write-Host "Failed to run go mod tidy: $_" -ForegroundColor Red
}

Write-Host "`n==== Import Fix Process Complete ====" -ForegroundColor Cyan
Write-Host "Backup of original files saved to: $backupDir" -ForegroundColor Green
