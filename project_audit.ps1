Write-Host "Starting Quant WebWork GO Project Audit..." -ForegroundColor Cyan

# Configuration
$requiredDirectories = @(
    "cmd/server",
    "cmd/webworks",
    "cmd/tools",
    "internal/api",
    "internal/bridge",
    "internal/core",
    "internal/security",
    "internal/storage",
    "internal/ui",
    "web/client",
    "deployments",
    "tests",
    "configs"
)

$keyFiles = @(
    "cmd/server/main.go",
    "internal/bridge/bridge.go",
    "internal/core/config/manager.go",
    "internal/api/routes/router.go",
    "web/client/src/index.tsx",
    "web/client/package.json",
    "deployments/Dockerfile",
    "docker-compose.yml",
    "go.mod"
)

$requiredPackages = @(
    "github.com/gorilla/mux",
    "github.com/gorilla/websocket",
    "github.com/prometheus/client_golang",
    "github.com/graphql-go/graphql",
    "github.com/google/uuid",
    "google.golang.org/grpc"
)

function Test-RequiredDirectories {
    Write-Host "`nChecking Required Directories..." -ForegroundColor Yellow
    $missingDirs = @()
    
    foreach ($dir in $requiredDirectories) {
        if (Test-Path $dir) {
            Write-Host "✓ $dir" -ForegroundColor Green
        } else {
            Write-Host "✗ $dir (Missing)" -ForegroundColor Red
            $missingDirs += $dir
        }
    }
    
    return $missingDirs
}

function Test-KeyFiles {
    Write-Host "`nChecking Key Files..." -ForegroundColor Yellow
    $missingFiles = @()
    
    foreach ($file in $keyFiles) {
        if (Test-Path $file) {
            Write-Host "✓ $file" -ForegroundColor Green
        } else {
            Write-Host "✗ $file (Missing)" -ForegroundColor Red
            $missingFiles += $file
        }
    }
    
    return $missingFiles
}

function Test-GoModDependencies {
    Write-Host "`nChecking Go Module Dependencies..." -ForegroundColor Yellow
    $missingDeps = @()
    
    if (Test-Path "go.mod") {
        $goModContent = Get-Content -Path "go.mod" -Raw
        
        foreach ($pkg in $requiredPackages) {
            if ($goModContent -match $pkg) {
                Write-Host "✓ $pkg" -ForegroundColor Green
            } else {
                Write-Host "✗ $pkg (Missing)" -ForegroundColor Red
                $missingDeps += $pkg
            }
        }
    } else {
        Write-Host "✗ go.mod file not found" -ForegroundColor Red
        $missingDeps = $requiredPackages
    }
    
    return $missingDeps
}

function Test-ImportErrors {
    Write-Host "`nScanning for Import Errors in Go Files..." -ForegroundColor Yellow
    
    $goFiles = Get-ChildItem -Path . -Filter "*.go" -Recurse | Where-Object { 
        -not $_.FullName.Contains("\vendor\") -and 
        -not $_.FullName.Contains("\.git\")
    }
    
    $filesWithErrors = @()
    $errorPattern = "github.com/IAM-timmy1t/Quant_WebWork_GO/QUANT_WW_GO"
    
    foreach ($file in $goFiles) {
        $content = Get-Content -Path $file.FullName -Raw
        if ($content -match $errorPattern) {
            Write-Host "✗ $($file.FullName) (Contains incorrect import paths)" -ForegroundColor Red
            $filesWithErrors += $file.FullName
        }
    }
    
    if ($filesWithErrors.Count -eq 0) {
        Write-Host "✓ No files with incorrect import paths found" -ForegroundColor Green
    }
    
    return $filesWithErrors
}

function Test-WebClientSetup {
    Write-Host "`nChecking Web Client Setup..." -ForegroundColor Yellow
    
    $webClientRoot = "web/client"
    $webClientErrors = @()
    
    if (Test-Path $webClientRoot) {
        # Check for package.json
        if (Test-Path "$webClientRoot/package.json") {
            Write-Host "✓ package.json exists" -ForegroundColor Green
            
            # Check for required dependencies
            $packageJson = Get-Content -Path "$webClientRoot/package.json" -Raw | ConvertFrom-Json
            $requiredNpmPackages = @("react", "typescript")
            
            foreach ($pkg in $requiredNpmPackages) {
                if ($packageJson.dependencies.PSObject.Properties.Name -contains $pkg -or 
                    $packageJson.devDependencies.PSObject.Properties.Name -contains $pkg) {
                    Write-Host "✓ npm package: $pkg" -ForegroundColor Green
                } else {
                    Write-Host "✗ npm package: $pkg (Missing)" -ForegroundColor Red
                    $webClientErrors += "Missing npm package: $pkg"
                }
            }
            
            # Check for source directory
            if (Test-Path "$webClientRoot/src") {
                Write-Host "✓ src directory exists" -ForegroundColor Green
            } else {
                Write-Host "✗ src directory missing" -ForegroundColor Red
                $webClientErrors += "Missing src directory in web client"
            }
        } else {
            Write-Host "✗ package.json missing" -ForegroundColor Red
            $webClientErrors += "Missing package.json in web client"
        }
    } else {
        Write-Host "✗ Web client directory not found" -ForegroundColor Red
        $webClientErrors += "Missing web client directory"
    }
    
    return $webClientErrors
}

function Test-DockerSetup {
    Write-Host "`nChecking Docker Setup..." -ForegroundColor Yellow
    
    $dockerErrors = @()
    
    # Check for Dockerfile
    if (Test-Path "Dockerfile" -or (Test-Path "deployments/Dockerfile")) {
        Write-Host "✓ Dockerfile exists" -ForegroundColor Green
    } else {
        Write-Host "✗ Dockerfile missing" -ForegroundColor Red
        $dockerErrors += "Missing Dockerfile"
    }
    
    # Check for docker-compose.yml
    if (Test-Path "docker-compose.yml") {
        Write-Host "✓ docker-compose.yml exists" -ForegroundColor Green
    } else {
        Write-Host "✗ docker-compose.yml missing" -ForegroundColor Red
        $dockerErrors += "Missing docker-compose.yml"
    }
    
    return $dockerErrors
}

function Test-TokenManagementFiles {
    Write-Host "`nChecking for Unnecessary Token Management Files..." -ForegroundColor Yellow
    
    $tokenFiles = Get-ChildItem -Path . -Recurse -Include "*token*" | Where-Object { 
        -not $_.FullName.Contains("\vendor\") -and 
        -not $_.FullName.Contains("\.git\")
    }
    
    if ($tokenFiles.Count -gt 0) {
        foreach ($file in $tokenFiles) {
            Write-Host "! $($file.FullName) (Potential unused token file)" -ForegroundColor Yellow
        }
    } else {
        Write-Host "✓ No unnecessary token management files found" -ForegroundColor Green
    }
    
    return $tokenFiles
}

# Run all checks
$missingDirs = Test-RequiredDirectories
$missingFiles = Test-KeyFiles
$missingDeps = Test-GoModDependencies
$filesWithImportErrors = Test-ImportErrors
$webClientErrors = Test-WebClientSetup
$dockerErrors = Test-DockerSetup
$tokenFiles = Test-TokenManagementFiles

# Generate summary report
Write-Host "`nAUDIT SUMMARY" -ForegroundColor Cyan
Write-Host "===============" -ForegroundColor Cyan

if ($missingDirs.Count -eq 0 -and 
    $missingFiles.Count -eq 0 -and 
    $missingDeps.Count -eq 0 -and 
    $filesWithImportErrors.Count -eq 0 -and 
    $webClientErrors.Count -eq 0 -and 
    $dockerErrors.Count -eq 0) {
    
    Write-Host "`nProject passes all checks!" -ForegroundColor Green
} else {
    Write-Host "`nProject has the following issues:" -ForegroundColor Yellow
    
    if ($missingDirs.Count -gt 0) {
        Write-Host "  - Missing directories: $($missingDirs.Count)" -ForegroundColor Red
    }
    
    if ($missingFiles.Count -gt 0) {
        Write-Host "  - Missing key files: $($missingFiles.Count)" -ForegroundColor Red
    }
    
    if ($missingDeps.Count -gt 0) {
        Write-Host "  - Missing dependencies: $($missingDeps.Count)" -ForegroundColor Red
    }
    
    if ($filesWithImportErrors.Count -gt 0) {
        Write-Host "  - Files with import errors: $($filesWithImportErrors.Count)" -ForegroundColor Red
        Write-Host "    ! Run .\fix_imports.ps1 to fix these issues" -ForegroundColor Yellow
    }
    
    if ($webClientErrors.Count -gt 0) {
        Write-Host "  - Web client issues: $($webClientErrors.Count)" -ForegroundColor Red
    }
    
    if ($dockerErrors.Count -gt 0) {
        Write-Host "  - Docker setup issues: $($dockerErrors.Count)" -ForegroundColor Red
    }
}

if ($tokenFiles.Count -gt 0) {
    Write-Host "`nFound $($tokenFiles.Count) potential unused token management files" -ForegroundColor Yellow
    Write-Host "   Consider reviewing these files to determine if they can be removed" -ForegroundColor Yellow
}

Write-Host "`nSUGGESTED NEXT STEPS:" -ForegroundColor Cyan
if ($filesWithImportErrors.Count -gt 0) {
    Write-Host "1. Run .\fix_imports.ps1 to fix import path issues" -ForegroundColor Yellow
}

if ($missingDeps.Count -gt 0) {
    Write-Host "2. Run 'go mod tidy' to update dependencies" -ForegroundColor Yellow
}

Write-Host "3. Run 'go build ./cmd/server' to verify the project builds successfully" -ForegroundColor Yellow
Write-Host "4. Run the application with 'go run ./cmd/server/main.go'" -ForegroundColor Yellow

Write-Host "`nAudit completed!" -ForegroundColor Cyan 