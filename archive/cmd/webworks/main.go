package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/api/rest"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/api/websocket"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/config"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/dashboard"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/monitoring"
	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/storage"
)

const (
	version = "0.1.0"
)

func main() {
	// Define command line flags
	var (
		showVersion = flag.Bool("version", false, "Show version information")
		configFile  = flag.String("config", "config.yaml", "Path to configuration file")
		logLevel    = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("Quant_WebWorks_GO version %s\n", version)
		os.Exit(0)
	}

	// Initialize logger
	logger := logrus.New()
	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		logger.Fatalf("Invalid log level: %v", err)
	}
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.JSONFormatter{})

	logger.WithField("version", version).Info("Starting Quant_WebWorks_GO")

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize tracing
	tp := initTracing(logger)
	defer tp.Shutdown(context.Background())

	// Initialize metrics
	mp := initMetrics(logger)
	defer mp.Shutdown(context.Background())

	// Initialize storage
	store, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		logger.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize monitoring
	resourceMonitor := monitoring.NewResourceMonitor(cfg.Monitoring)
	securityMonitor := monitoring.NewSecurityMonitor(cfg.Security)

	// Initialize dashboard service
	dashboardService := dashboard.NewService(resourceMonitor, securityMonitor, store, tp)
	dashboardService.Start(context.Background())
	defer dashboardService.Stop()

	// Initialize WebSocket hub
	wsHub := websocket.NewDashboardHub(dashboardService)
	wsHub.Start(context.Background())
	defer wsHub.Stop()

	// Initialize router
	router := mux.NewRouter()

	// Register REST handlers
	restHandler := rest.NewDashboardHandler(dashboardService)
	restHandler.RegisterRoutes(router)

	// Register WebSocket handler
	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWS(wsHub, w, r)
	})

	// Register Prometheus metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// Configure CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   cfg.Server.CorsAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:          300,
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.Server.ListenAddress,
		Handler:      corsHandler.Handler(router),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("address", cfg.Server.ListenAddress).Info("Starting HTTP server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("Server shutdown error: %v", err)
	}

	logger.Info("Server stopped")
}

func initTracing(logger *logrus.Logger) *trace.TracerProvider {
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	logger.Info("Tracing initialized")
	return tp
}

func initMetrics(logger *logrus.Logger) *metric.MeterProvider {
	exporter, err := prometheus.New()
	if err != nil {
		logger.Fatalf("Failed to create Prometheus exporter: %v", err)
	}

	mp := metric.NewMeterProvider(metric.WithReader(exporter))
	otel.SetMeterProvider(mp)
	logger.Info("Metrics initialized")
	return mp
}

