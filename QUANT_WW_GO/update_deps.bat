@echo off
echo [CSM-PROC] QUANT WebWorks GO - Dependency Update Script
echo ------------------------------------------------------

REM Check Go installation
where go >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo [CSM-ERR] Go is not installed or not in PATH
    exit /b 1
)

echo [CSM-INFO] Using Go version:
go version

REM Create backups
echo [CSM-PROC] Creating backup of module files
copy go.mod go.mod.backup
if exist go.sum copy go.sum go.sum.backup

REM Add required dependencies for monitoring
echo [CSM-PROC] Adding required dependencies...

echo [CSM-INFO] Adding Prometheus client
go get -u github.com/prometheus/client_golang@v1.18.0

echo [CSM-INFO] Adding gRPC Gateway
go get -u github.com/grpc-ecosystem/grpc-gateway/v2@v2.19.1

echo [CSM-INFO] Adding Google gRPC
go get -u google.golang.org/grpc@v1.62.0

echo [CSM-INFO] Adding error details package
go get -u google.golang.org/genproto/googleapis/rpc/errdetails@latest

REM Run go mod tidy
echo [CSM-PROC] Running go mod tidy...
go mod tidy

REM Download all dependencies
echo [CSM-PROC] Downloading all dependencies...
go mod download all

echo [CSM-OK] Dependencies updated successfully
echo ------------------------------------------------------
