@echo off
echo [CSM-PROC] QUANT WebWorks GO - Monitoring Verification Script
echo =========================================================

REM Check Docker installation
where docker >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo [CSM-ERR] Docker is not installed or not in PATH
    exit /b 1
)

echo [CSM-INFO] Using Docker version:
docker --version

REM Check Docker Compose installation
where docker-compose >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo [CSM-ERR] Docker Compose is not installed or not in PATH. If using Docker Desktop, this should be included.
    exit /b 1
)

echo [CSM-INFO] Using Docker Compose version:
docker-compose --version

REM Check if monitoring files exist
echo [CSM-PROC] Verifying monitoring configuration files...

if not exist "monitoring\prometheus\prometheus.yml" (
    echo [CSM-ERR] Missing Prometheus configuration file at monitoring\prometheus\prometheus.yml
    exit /b 1
) else (
    echo [CSM-OK] Prometheus configuration file exists
)

if not exist "monitoring\grafana\provisioning\dashboards\bridge-dashboard.json" (
    echo [CSM-ERR] Missing Grafana dashboard at monitoring\grafana\provisioning\dashboards\bridge-dashboard.json
    exit /b 1
) else (
    echo [CSM-OK] Grafana dashboard file exists
)

if not exist "monitoring\grafana\provisioning\datasources\prometheus.yml" (
    echo [CSM-ERR] Missing Grafana datasource configuration at monitoring\grafana\provisioning\datasources\prometheus.yml
    exit /b 1
) else (
    echo [CSM-OK] Grafana datasource configuration exists
)

REM Create dashboard provisioning file if not exists
if not exist "monitoring\grafana\provisioning\dashboards\dashboard.yml" (
    echo [CSM-PROC] Creating Grafana dashboard provisioning file...
    mkdir "monitoring\grafana\provisioning\dashboards" 2>nul
    
    echo apiVersion: 1 > monitoring\grafana\provisioning\dashboards\dashboard.yml
    echo. >> monitoring\grafana\provisioning\dashboards\dashboard.yml
    echo providers: >> monitoring\grafana\provisioning\dashboards\dashboard.yml
    echo - name: 'default' >> monitoring\grafana\provisioning\dashboards\dashboard.yml
    echo   orgId: 1 >> monitoring\grafana\provisioning\dashboards\dashboard.yml
    echo   folder: '' >> monitoring\grafana\provisioning\dashboards\dashboard.yml
    echo   type: file >> monitoring\grafana\provisioning\dashboards\dashboard.yml
    echo   disableDeletion: false >> monitoring\grafana\provisioning\dashboards\dashboard.yml
    echo   editable: true >> monitoring\grafana\provisioning\dashboards\dashboard.yml
    echo   options: >> monitoring\grafana\provisioning\dashboards\dashboard.yml
    echo     path: /etc/grafana/provisioning/dashboards >> monitoring\grafana\provisioning\dashboards\dashboard.yml

    echo [CSM-OK] Created Grafana dashboard provisioning file
)

REM Check Docker Compose file
echo [CSM-PROC] Verifying Docker Compose configuration...
if not exist "docker-compose.yml" (
    echo [CSM-ERR] Missing docker-compose.yml file
    exit /b 1
) else (
    echo [CSM-OK] Docker Compose file exists
)

REM Check if Docker services are running
echo [CSM-PROC] Checking if Docker services are running...
docker ps | findstr "prometheus" >nul
if %ERRORLEVEL% equ 0 (
    echo [CSM-INFO] Prometheus service is already running
) else (
    echo [CSM-INFO] Prometheus service is not running
)

docker ps | findstr "grafana" >nul
if %ERRORLEVEL% equ 0 (
    echo [CSM-INFO] Grafana service is already running
) else (
    echo [CSM-INFO] Grafana service is not running
)

REM Offer to start services
echo.
echo [CSM-PROC] Next steps:
echo 1. To start all services:
echo    docker-compose up -d
echo.
echo 2. To start only monitoring services:
echo    docker-compose up -d prometheus grafana
echo.
echo 3. To verify Prometheus is working:
echo    curl http://localhost:9091/api/v1/status/buildinfo
echo.
echo 4. To verify Grafana is working:
echo    curl http://localhost:3001/api/health
echo.
echo 5. Once running, access the following URLs:
echo    - Prometheus: http://localhost:9091
echo    - Grafana: http://localhost:3001 (login with admin/admin)
echo.
echo 6. To check for metrics from the Bridge Module:
echo    curl http://localhost:8080/metrics
echo.

echo [CSM-PROC] Monitoring verification completed
echo =========================================================
