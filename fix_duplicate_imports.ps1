# Fix Duplicate QUANT_WW_GO Segments in Import Paths
Write-Host "==== QUANT WebWork GO Import Path Fix ====" -ForegroundColor Cyan
Write-Host "This script removes duplicate QUANT_WW_GO segments in import paths" -ForegroundColor Cyan

$projectDir = "Z:\.CodingProjects\GitHub_Repos\Quant_WebWork_GO\QUANT_WW_GO"
$backupDir = "Z:\.CodingProjects\GitHub_Repos\Quant_WebWork_GO\import_fix_backup_$(Get-Date -Format 'yyyyMMdd_HHmmss')"

# Create backup directory
New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
Write-Host "Created backup directory: $backupDir" -ForegroundColor Green

# 1. Fix duplicate QUANT_WW_GO segments in import paths
Write-Host "`n[STEP 1] Fixing duplicate QUANT_WW_GO segments in import paths..." -ForegroundColor Cyan
$goFiles = Get-ChildItem -Path $projectDir -Filter "*.go" -Recurse
Write-Host "Found $($goFiles.Count) Go files to check for imports" -ForegroundColor White

$changedFiles = 0
$duplicatePattern = "github.com/IAM-timmy1t/Quant_WebWork_GO/QUANT_WW_GO/QUANT_WW_GO/"
$correctPattern = "github.com/IAM-timmy1t/Quant_WebWork_GO/"

foreach ($file in $goFiles) {
    # Backup the file first
    Copy-Item -Path $file.FullName -Destination "$backupDir\$($file.Name).backup" -Force
    
    $content = Get-Content -Path $file.FullName -Raw
    $originalContent = $content
    
    # Check if the file contains the duplicate pattern
    if ($content -match [regex]::Escape($duplicatePattern)) {
        $content = $content -replace [regex]::Escape($duplicatePattern), $correctPattern
        Write-Host "  - Fixed imports in: $($file.FullName)" -ForegroundColor Yellow
        
        # Also fix any single duplicate
        $singleDuplicatePattern = "github.com/IAM-timmy1t/Quant_WebWork_GO/QUANT_WW_GO/"
        if ($content -match [regex]::Escape($singleDuplicatePattern)) {
            $content = $content -replace [regex]::Escape($singleDuplicatePattern), $correctPattern
            Write-Host "    - Also fixed single duplicate in: $($file.FullName)" -ForegroundColor Yellow
        }
        
        # Write the fixed content back to the file
        Set-Content -Path $file.FullName -Value $content
        $changedFiles++
    }
}

Write-Host "Fixed imports in $changedFiles files" -ForegroundColor Green

# 2. Run go mod tidy to update dependencies
Write-Host "`n[STEP 2] Running 'go mod tidy' to update dependencies..." -ForegroundColor Cyan
try {
    Set-Location -Path $projectDir
    $output = & go mod tidy 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "go mod tidy completed successfully!" -ForegroundColor Green
    } else {
        Write-Host "Warning: go mod tidy output:" -ForegroundColor Yellow
        Write-Host $output
    }
} catch {
    Write-Host "Error running go mod tidy: $_" -ForegroundColor Red
}

# 3. Try to build the project
Write-Host "`n[STEP 3] Attempting to build the project..." -ForegroundColor Cyan
try {
    Set-Location -Path $projectDir
    $output = & go build ./... 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Build succeeded!" -ForegroundColor Green
    } else {
        Write-Host "Warning: build output:" -ForegroundColor Yellow
        Write-Host $output
    }
} catch {
    Write-Host "Error building project: $_" -ForegroundColor Red
}

Write-Host "`n==== Import Path Fix Complete ====" -ForegroundColor Cyan
Write-Host "Backup of original files saved to: $backupDir" -ForegroundColor Green
Write-Host "`nNext steps if you still have issues:" -ForegroundColor Yellow
Write-Host "1. Check if there are any remaining incorrect import paths" -ForegroundColor White
Write-Host "2. Verify that all required packages exist in your project" -ForegroundColor White
Write-Host "3. Run 'go build -v ./...' to see detailed compilation errors" -ForegroundColor White
