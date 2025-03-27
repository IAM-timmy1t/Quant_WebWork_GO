// main.go - Quant WebWork GO Server
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/api/rest"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/bridge"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/config"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/discovery"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/metrics"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/security"
	"go.uber.org/zap"
)

// Application constants
const (
	serviceName    = "quant-webwork-go"
	serviceVersion = "1.2.1"
)

// Command-line flags
var (
	configPath string
	devMode    bool
	logLevel   string
)

func init() {
	flag.StringVar(&configPath, "config", "./config/config.yaml", "Path to configuration file")
	flag.BoolVar(&devMode, "dev", false, "Run in development mode")
	flag.StringVar(&logLevel, "log-level", "info", "Logging level (debug, info, warn, error)")
}

func main() {
	flag.Parse()

	// Initialize logger
	var logger *zap.Logger
	var err error

	if devMode {
		// Development logger with colorized output and more verbose logging
		zapConfig := zap.NewDevelopmentConfig()
		if logLevel != "" {
			zapConfig.Level.UnmarshalText([]byte(logLevel))
		}
		logger, err = zapConfig.Build()
	} else {
		// Production logger with structured JSON output
		zapConfig := zap.NewProductionConfig()
		if logLevel != "" {
			zapConfig.Level.UnmarshalText([]byte(logLevel))
		}
		logger, err = zapConfig.Build()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	sugar.Infow("Starting Quant WebWork GO Server",
		"version", serviceVersion,
		"environment", getEnvironmentName(devMode),
		"configPath", configPath)

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		sugar.Fatalw("Failed to load configuration", "error", err)
	}

	// Apply environment-based security configuration
	envType := security.GetEnvironmentType()
	securityConfig := security.GetSecurityConfig(sugar)
	sugar.Infow("Security configuration loaded",
		"environment", envType,
		"securityLevel", securityConfig.RateLimitingLevel,
		"authRequired", securityConfig.AuthRequired)

	// Validate security for production environments
	if envType == security.EnvProduction && !devMode {
		if err := security.ValidateProductionSecurity(securityConfig); err != nil {
			sugar.Fatalw("Security validation failed for production", "error", err)
		}
	}

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector(cfg.Monitoring.Metrics, sugar)

	// Start metrics server if enabled
	if cfg.Monitoring.Metrics.Enabled {
		metricsServer := metrics.NewServer(cfg.Monitoring.Metrics, metricsCollector, sugar)
		go func() {
			if err := metricsServer.Start(); err != nil {
				sugar.Errorw("Metrics server failed", "error", err)
			}
		}()
	}

	// Initialize discovery service
	discoveryService, err := discovery.NewService(cfg.Bridge.Discovery, sugar)
	if err != nil {
		sugar.Fatalw("Failed to initialize discovery service", "error", err)
	}

	// Setup bridge manager
	bridgeManagerConfig := &bridge.ManagerConfig{
		DefaultTimeout:      time.Second * 30,
		DefaultRetryCount:   3,
		DefaultRetryDelay:   time.Second * 5,
		HealthCheckInterval: time.Second * 60,
		EventBufferSize:     100,
		MetricsEnabled:      true,
	}
	bridgeManager := bridge.NewManager(bridgeManagerConfig)
	bridgeManager.SetLogger(sugar)
	bridgeManager.SetMetricsCollector(metricsCollector)

	// Register protocols based on config
	for _, protocol := range cfg.Bridge.Protocols {
		// Simplified - in a real implementation we would create actual protocol adapters
		sugar.Infow("Registering protocol", "protocol", protocol)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start bridge manager
	sugar.Info("Starting bridge manager")
	if err := bridgeManager.Start(ctx); err != nil {
		sugar.Fatalw("Failed to start bridge manager", "error", err)
	}
	defer func() {
		sugar.Info("Stopping bridge manager")
		if err := bridgeManager.Stop(); err != nil {
			sugar.Errorw("Error stopping bridge manager", "error", err)
		}
	}()

	// Create security monitor with the correct config type
	monitorConfig := security.DefaultConfig()
	securityMonitor, err := security.NewMonitor(monitorConfig)
	if err != nil {
		sugar.Fatalw("Failed to initialize security monitor", "error", err)
	}
	defer securityMonitor.Close()

	// Setup API router
	sugar.Info("Initializing API router")
	router := rest.NewRouter(cfg, sugar, metricsCollector)

	// Configure HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.ShutdownTimeout,
	}

	// Start server in a goroutine
	go func() {
		sugar.Infow("Starting HTTP server", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sugar.Fatalw("Server failed", "error", err)
		}
	}()

	// Register service with discovery service if enabled
	if cfg.Bridge.Discovery.Enabled {
		svcID := registerWithDiscovery(discoveryService, cfg, sugar)
		defer func() {
			if err := discoveryService.UnregisterService(svcID); err != nil {
				sugar.Errorw("Failed to unregister service", "error", err)
			}
		}()
	}

	// Handle graceful shutdown
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
	<-stopCh
	sugar.Info("Received shutdown signal")

	// Create context with timeout for shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	// Shutdown HTTP server
	sugar.Info("Shutting down HTTP server")
	if err := server.Shutdown(shutdownCtx); err != nil {
		sugar.Errorw("Server forced to shutdown", "error", err)
	}

	sugar.Info("Server gracefully stopped")
}

// getEnvironmentName returns a human-readable environment name
func getEnvironmentName(devMode bool) string {
	if devMode {
		return "development"
	}

	env := os.Getenv("QUANT_ENV")
	switch env {
	case "prod", "production":
		return "production"
	case "staging":
		return "staging"
	case "test", "testing":
		return "testing"
	default:
		return "production" // Default to production if not specified
	}
}

// registerWithDiscovery registers the service with the discovery service
func registerWithDiscovery(discoveryService *discovery.BridgeDiscovery, cfg *config.Config, logger *zap.SugaredLogger) string {
	svc := &discovery.Service{
		ID:          fmt.Sprintf("%s-%d", serviceName, os.Getpid()),
		Name:        serviceName,
		Protocol:    "http",
		Host:        cfg.Server.Host,
		Port:        cfg.Server.Port,
		HealthCheck: "/health",
		Metadata: map[string]string{
			"version": serviceVersion,
		},
		Status: "available",
	}

	if err := discoveryService.RegisterService(svc); err != nil {
		logger.Errorw("Failed to register with discovery service", "error", err)
		return ""
	}

	logger.Infow("Registered with discovery service", "serviceID", svc.ID)
	return svc.ID
}
