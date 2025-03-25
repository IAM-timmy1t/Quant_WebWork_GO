# Update Dependencies PowerShell Script for QUANT_WW_GO
# =======================================================

Write-Host "QUANT WebWorks GO - Dependency Update Script" -ForegroundColor Green
Write-Host "===============================================" -ForegroundColor Green
Write-Host ""

# Set the error action preference to stop on errors
$ErrorActionPreference = "Stop"

# Navigate to the project directory
$projectDir = $PSScriptRoot
Write-Host "Project directory: $projectDir" -ForegroundColor Cyan

# Function to check if command exists
function Test-CommandExists {
    param ($command)
    $exists = $null -ne (Get-Command $command -ErrorAction SilentlyContinue)
    return $exists
}

# Check Go installation
if (-not (Test-CommandExists "go")) {
    Write-Host "ERROR: Go is not installed or not in PATH" -ForegroundColor Red
    exit 1
}

# Display Go version
Write-Host "Using Go version:" -ForegroundColor Cyan
go version

# Add required gRPC-related dependencies
Write-Host "Adding required dependencies..." -ForegroundColor Cyan
$deps = @(
    "google.golang.org/grpc",
    "google.golang.org/genproto",
    "github.com/grpc-ecosystem/grpc-gateway/v2",
    "github.com/prometheus/client_golang",
    "github.com/prometheus/client_golang/prometheus/promhttp",
    "google.golang.org/grpc/status",
    "google.golang.org/grpc/codes",
    "google.golang.org/grpc/credentials",
    "google.golang.org/grpc/web"
)

foreach ($dep in $deps) {
    Write-Host "Adding dependency: $dep" -ForegroundColor Yellow
    try {
        go get -u $dep
    }
    catch {
        Write-Host "Warning: Could not add $dep. Error: $_" -ForegroundColor Yellow
        # Continue with other dependencies
    }
}

# Run go mod tidy to clean up dependencies
Write-Host "Running go mod tidy..." -ForegroundColor Cyan
try {
    go mod tidy
}
catch {
    Write-Host "Warning: go mod tidy reported issues: $_" -ForegroundColor Yellow
}

# Verify go.mod and go.sum files
if (Test-Path "$projectDir/go.mod") {
    Write-Host "go.mod file exists." -ForegroundColor Green
}
else {
    Write-Host "ERROR: go.mod file not found." -ForegroundColor Red
}

if (Test-Path "$projectDir/go.sum") {
    Write-Host "go.sum file exists." -ForegroundColor Green
}
else {
    Write-Host "WARNING: go.sum file not found." -ForegroundColor Yellow
}

# Update go.mod to handle import path issues
Write-Host "Updating module paths in go.mod..." -ForegroundColor Cyan

$goModContent = Get-Content "$projectDir/go.mod" -Raw
$moduleName = ($goModContent -split "\n" | Select-String -Pattern "^module\s+(.+)$").Matches.Groups[1].Value

Write-Host "Module name is: $moduleName" -ForegroundColor Cyan

# Check for potential import path issues
if ($moduleName -ne "github.com/timot/Quant_WebWork_GO/QUANT_WW_GO") {
    Write-Host "Fixing module name..." -ForegroundColor Yellow
    $goModContent = $goModContent -replace "^module\s+.+$", "module github.com/timot/Quant_WebWork_GO/QUANT_WW_GO"
    $goModContent | Set-Content "$projectDir/go.mod"
}

# Check for errdetails import
if (-not ($goModContent -like "*google.golang.org/genproto/googleapis/rpc/errdetails*")) {
    Write-Host "Adding errdetails dependency..." -ForegroundColor Yellow
    go get -u google.golang.org/genproto/googleapis/rpc/errdetails
}

# Run go mod tidy one more time
Write-Host "Running final go mod tidy..." -ForegroundColor Cyan
try {
    go mod tidy
}
catch {
    Write-Host "Warning: Final go mod tidy reported issues: $_" -ForegroundColor Yellow
}

# Install required tools
Write-Host "Installing required tools..." -ForegroundColor Cyan
$tools = @(
    "google.golang.org/protobuf/cmd/protoc-gen-go",
    "google.golang.org/grpc/cmd/protoc-gen-go-grpc",
    "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway"
)

foreach ($tool in $tools) {
    Write-Host "Installing tool: $tool" -ForegroundColor Yellow
    try {
        go install $tool@latest
    }
    catch {
        Write-Host "Warning: Could not install $tool. Error: $_" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "==================================" -ForegroundColor Green
Write-Host "Dependency update process complete" -ForegroundColor Green
Write-Host "==================================" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "1. Check that all imports resolve correctly" -ForegroundColor Cyan
Write-Host "2. Fix any specific import issues in the code" -ForegroundColor Cyan
Write-Host "3. Run 'go build ./...' to verify the build" -ForegroundColor Cyan
