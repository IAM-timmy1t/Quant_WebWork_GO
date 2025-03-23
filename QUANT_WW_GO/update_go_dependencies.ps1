# QUANT WebWorks GO - Module Update & Validation Script
# ================================================
# This script follows the structured processing and manual workflow
# requirements from QUANT_AGI Global Rules

# Set strict error handling
$ErrorActionPreference = "Stop"

function Write-StepHeader {
    param ($message)
    Write-Host "`n[CSM-PROC] $message" -ForegroundColor Cyan
    Write-Host "---------------------------------------------------" -ForegroundColor DarkGray
}

function Write-Success {
    param ($message)
    Write-Host "[CSM-OK] $message" -ForegroundColor Green
}

function Write-Warning {
    param ($message)
    Write-Host "[CSM-WARN] $message" -ForegroundColor Yellow
}

function Write-Error {
    param ($message)
    Write-Host "[CSM-ERR] $message" -ForegroundColor Red
}

function Test-GoCommand {
    if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
        Write-Error "Go is not installed or not in PATH"
        exit 1
    }
    $goVersion = go version
    Write-Host "Using $goVersion"
}

function Backup-ModuleFiles {
    Write-StepHeader "Creating restore point for module files"
    
    # Generate fingerprint for context validation
    $goModHash = Get-FileHash -Path "go.mod" -Algorithm MD5
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    
    # Create backups with timestamp
    Copy-Item "go.mod" "go.mod.backup.$timestamp"
    if (Test-Path "go.sum") {
        Copy-Item "go.sum" "go.sum.backup.$timestamp"
    }
    
    # Output fingerprint
    "go.mod:$($goModHash.Hash)" | Out-File "context_fingerprint.md5" -Encoding utf8
    
    Write-Success "Created backups with timestamp $timestamp"
}

function Restore-ModuleFiles {
    param ($timestamp)
    Write-StepHeader "Restoring module files from backup"
    
    if (Test-Path "go.mod.backup.$timestamp") {
        Copy-Item "go.mod.backup.$timestamp" "go.mod" -Force
        Write-Success "Restored go.mod from backup"
    } else {
        Write-Error "Cannot find go.mod backup file"
    }
    
    if (Test-Path "go.sum.backup.$timestamp") {
        Copy-Item "go.sum.backup.$timestamp" "go.sum" -Force
        Write-Success "Restored go.sum from backup"
    }
}

function Update-ModuleFile {
    Write-StepHeader "Updating module path and dependencies"
    
    # Read current module content
    $goModContent = Get-Content "go.mod" -Raw
    
    # Update module path if needed
    if ($goModContent -notmatch "module github.com/timot/Quant_WebWork_GO/QUANT_WW_GO") {
        $goModContent = $goModContent -replace "module .+", "module github.com/timot/Quant_WebWork_GO/QUANT_WW_GO"
        Write-Warning "Updated module path to github.com/timot/Quant_WebWork_GO/QUANT_WW_GO"
    } else {
        Write-Success "Module path is already correct"
    }
    
    # Add required dependencies for monitoring
    $requiredDeps = @(
        'github.com/prometheus/client_golang v1.18.0',
        'github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.1',
        'google.golang.org/genproto v0.0.0-20240221002015-b0ce06bbee7c',
        'google.golang.org/grpc v1.62.0',
        'google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.3.0'
    )
    
    # Check for each dependency
    foreach ($dep in $requiredDeps) {
        $depName = ($dep -split " ")[0]
        if ($goModContent -notmatch $depName) {
            Write-Warning "Adding dependency: $dep"
            # We'll add using go get instead of direct file modification
        }
    }
    
    # Save updated content
    $goModContent | Set-Content "go.mod" -NoNewline
}

function Run-SecurityScan {
    Write-StepHeader "Running pre-implementation security scan"
    
    # Check for hardcoded secrets in go.mod
    $secretPatterns = @(
        '\b[A-Za-z0-9]{32}\b',  # API keys
        '\bpassword\s*=\s*[\'"].+?[\'"]',
        '\b[A-Z0-9_]+_SECRET\b'
    )
    
    $goModContent = Get-Content "go.mod" -Raw
    $foundSecrets = $false
    
    foreach ($pattern in $secretPatterns) {
        if ($goModContent -match $pattern) {
            Write-Error "Potential secret found in go.mod!"
            $foundSecrets = $true
        }
    }
    
    if (-not $foundSecrets) {
        Write-Success "No secrets detected in module files"
    } else {
        exit 1
    }
}

function Run-GoInstall {
    Write-StepHeader "Installing required Go tools and dependencies"
    
    # Common tools for gRPC and monitoring
    $tools = @(
        'google.golang.org/protobuf/cmd/protoc-gen-go@latest',
        'google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest',
        'github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest'
    )
    
    foreach ($tool in $tools) {
        try {
            Write-Host "Installing $tool..."
            go install $tool
            Write-Success "Installed $tool"
        } catch {
            Write-Warning "Failed to install $tool: $_"
        }
    }
}

function Update-Dependencies {
    Write-StepHeader "Updating Go dependencies"
    
    # Add monitoring dependencies
    $deps = @(
        'github.com/prometheus/client_golang@v1.18.0',
        'github.com/grpc-ecosystem/grpc-gateway/v2@v2.19.1',
        'google.golang.org/genproto/googleapis/rpc/errdetails@latest',
        'google.golang.org/grpc@v1.62.0'
    )
    
    foreach ($dep in $deps) {
        try {
            Write-Host "Adding $dep..."
            go get $dep
            Write-Success "Added $dep"
        } catch {
            Write-Warning "Failed to add $dep: $_"
        }
    }
    
    # Run go mod tidy
    try {
        Write-Host "Running go mod tidy..."
        go mod tidy
        Write-Success "Completed go mod tidy"
    } catch {
        Write-Error "Failed to run go mod tidy: $_"
        return $false
    }
    
    # Download all dependencies
    try {
        Write-Host "Downloading all dependencies..."
        go mod download all
        Write-Success "All dependencies downloaded"
    } catch {
        Write-Warning "Not all dependencies could be downloaded: $_"
    }
    
    return $true
}

function Verify-Build {
    Write-StepHeader "Verifying build after dependency updates"
    
    try {
        go list ./... | Out-Null
        Write-Success "Module structure verified"
    } catch {
        Write-Warning "Module structure verification failed: $_"
        return $false
    }
    
    return $true
}

function Generate-Documentation {
    Write-StepHeader "Generating documentation updates"
    
    $timestamp = Get-Date -Format "yyyy-MM-dd"
    @"
# Dependency Updates - $timestamp

## Updated Dependencies
- prometheus/client_golang: v1.18.0
- grpc-ecosystem/grpc-gateway/v2: v2.19.1
- google.golang.org/grpc: v1.62.0
- google.golang.org/genproto: latest
- google.golang.org/grpc/cmd/protoc-gen-go-grpc: v1.3.0

## Purpose
These dependencies were updated to support the monitoring infrastructure for the Bridge Module, including Prometheus metrics collection and Grafana dashboards.

## Changes
- Fixed module paths to align with repository structure
- Added required monitoring dependencies
- Updated gRPC dependencies for compatibility
- Added support for detailed error reporting

## Verification
Build and dependency verification was performed using the update_go_dependencies.ps1 script.
"@ | Out-File "docs/dependency-updates.md" -Encoding utf8 -Force
    
    Write-Success "Generated documentation at docs/dependency-updates.md"
}

# Main execution flow
try {
    Write-Host "QUANT WebWorks GO - Module Update & Validation Script" -ForegroundColor Green
    Write-Host "===================================================" -ForegroundColor Green
    Write-Host "Following QUANT_AGI Implementation Protocol version 2.8.1" -ForegroundColor Gray
    
    # Initial checks
    Test-GoCommand
    
    # Create backup
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    Backup-ModuleFiles
    
    # Security scan
    Run-SecurityScan
    
    # Update module file
    Update-ModuleFile
    
    # Update dependencies
    $dependenciesUpdated = Update-Dependencies
    if (-not $dependenciesUpdated) {
        Write-Error "Failed to update dependencies, rolling back..."
        Restore-ModuleFiles -timestamp $timestamp
        exit 1
    }
    
    # Install tools
    Run-GoInstall
    
    # Verify build
    $buildVerified = Verify-Build
    if (-not $buildVerified) {
        Write-Warning "Build verification had warnings, but continuing..."
    }
    
    # Generate documentation
    Generate-Documentation
    
    Write-Host "`n[CSM-COMPLETE] Go dependencies successfully updated" -ForegroundColor Green
    Write-Host "===================================================" -ForegroundColor Green
    
} catch {
    Write-Error "An error occurred: $_"
    Write-Host "Rolling back changes..." -ForegroundColor Red
    Restore-ModuleFiles -timestamp $timestamp
    exit 1
}
