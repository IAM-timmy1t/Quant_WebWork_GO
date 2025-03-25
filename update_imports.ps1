# Script to update import paths in Go files
# Updates from github.com/timot/Quant_WebWork_GO to github.com/IAM-timmy1t/Quant_WebWork_GO

$oldImportPath = "github.com/timot/Quant_WebWork_GO"
$newImportPath = "github.com/IAM-timmy1t/Quant_WebWork_GO"
$projectDir = "Z:\.CodingProjects\GitHub_Repos\Quant_WebWork_GO\QUANT_WW_GO"

# Find all Go files in the project
$goFiles = Get-ChildItem -Path $projectDir -Filter "*.go" -Recurse

$changedFiles = 0
$totalImportChanges = 0

foreach ($file in $goFiles) {
    $content = Get-Content -Path $file.FullName -Raw
    
    # Check if the file contains the old import path
    if ($content -match [regex]::Escape($oldImportPath)) {
        # Replace old import path with new import path
        $newContent = $content -replace [regex]::Escape($oldImportPath), $newImportPath
        
        # Count the number of replacements
        $importChanges = ([regex]::Matches($content, [regex]::Escape($oldImportPath))).Count
        $totalImportChanges += $importChanges
        
        # Save the changes
        Set-Content -Path $file.FullName -Value $newContent
        
        $changedFiles++
        Write-Host "Updated $($file.FullName) with $importChanges import changes"
    }
}

Write-Host "`nImport path update complete"
Write-Host "Changed $changedFiles files with a total of $totalImportChanges import references"
