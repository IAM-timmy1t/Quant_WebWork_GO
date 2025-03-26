param (
    [string]$ModulePath = "github.com/IAM-timmy1t/Quant_WebWork_GO"
)

Write-Host "🔍 Fixing import paths across the project..." -ForegroundColor Cyan
Write-Host "Module path: $ModulePath" -ForegroundColor Yellow

# Find all .go files in the project
$goFiles = Get-ChildItem -Path . -Filter "*.go" -Recurse | Where-Object { 
    -not $_.FullName.Contains("\vendor\") -and 
    -not $_.FullName.Contains("\.git\") 
}

$incorrectImportPattern = "${ModulePath}/QUANT_WW_GO/QUANT_WW_GO"
$correctImportPath = "${ModulePath}"

Write-Host "Found $($goFiles.Count) Go files to process" -ForegroundColor Green
$fixedCount = 0

foreach ($file in $goFiles) {
    $originalContent = Get-Content -Path $file.FullName -Raw
    $newContent = $originalContent

    # Fix import statements with redundant paths
    $newContent = $newContent -replace $incorrectImportPattern, $correctImportPath
    
    if ($newContent -ne $originalContent) {
        $fixedCount++
        
        # Write the fixed content back to the file
        Set-Content -Path $file.FullName -Value $newContent -NoNewline
        Write-Host "✅ Fixed imports in: $($file.FullName)" -ForegroundColor Green
    }
}

# Run go mod tidy to clean up dependencies
Write-Host "🧹 Running go mod tidy to clean up dependencies..." -ForegroundColor Cyan
go mod tidy

# Verify the go.mod file
Write-Host "🔍 Verifying go.mod file..." -ForegroundColor Cyan
go mod verify

Write-Host "✅ Import paths fixed in $fixedCount files!" -ForegroundColor Green
Write-Host "🎉 Project import paths have been updated successfully!" -ForegroundColor Cyan 