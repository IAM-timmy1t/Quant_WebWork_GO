// main.go - Quant WebWork GO Server
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/timot/Quant_WebWork_GO/QUANT_WW_GO/internal/core/config"
	"github.com/timot/Quant_WebWork_GO/QUANT_WW_GO/internal/core/discovery"
	"github.com/timot/Quant_WebWork_GO/QUANT_WW_GO/internal/core/metrics"
	"github.com/timot/Quant_WebWork_GO/QUANT_WW_GO/internal/security"
)

const (
	serviceName    = "quant-webwork-server"
	serviceVersion = "1.0.0"
)

func main() {
	// Initialize logger
	logger := log.New(os.Stdout, "[QUANT] ", log.LstdFlags|log.Lshortfile)
	logger.Println("Starting Quant WebWork GO Server...")

	// Load configuration
	configPath := getConfigPath()
	configManager, err := setupConfiguration(configPath)
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Create service context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize core components
	metricsCollector := setupMetrics(configManager)
	securityMonitor := setupSecurity(configManager, metricsCollector)
	discoveryService := setupDiscovery(configManager, serviceName, serviceVersion)

	// Start API server
	serverPort, _ := configManager.GetInt("server.port")
	if serverPort == 0 {
		serverPort = 8080 // Default port
	}
	
	server := setupServer(serverPort, metricsCollector, securityMonitor)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		logger.Println("Shutting down server...")
		
		// Create shutdown context with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		
		// Shutdown HTTP server
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Printf("Server shutdown error: %v", err)
		}
		
		// Deregister from service discovery
		if discoveryService != nil {
			discoveryService.Deregister(serviceName)
		}
		
		// Signal the main context to cancel
		cancel()
	}()

	// Start the server
	logger.Printf("Server listening on port %d", serverPort)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Server error: %v", err)
	}

	// Wait for context cancellation (from the shutdown goroutine)
	<-ctx.Done()
	logger.Println("Server shutdown complete")
}

func getConfigPath() string {
	// Check environment variable first
	configPath := os.Getenv("QUANT_CONFIG_PATH")
	if configPath != "" {
		return configPath
	}
	
	// Check command line arguments
	if len(os.Args) > 1 {
		return os.Args[1]
	}
	
	// Default config path
	return "./config/config.json"
}

func setupConfiguration(configPath string) (*config.Manager, error) {
	manager := config.NewManager()
	
	// Create file provider
	fileProvider, err := config.NewFileProvider(configPath, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create config provider: %w", err)
	}
	
	// Register provider
	if err := manager.RegisterProvider("file", fileProvider); err != nil {
		return nil, fmt.Errorf("failed to register config provider: %w", err)
	}
	
	// Load configuration
	if err := fileProvider.Load(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	
	return manager, nil
}

func setupMetrics(configManager *config.Manager) *metrics.Collector {
	// Create metrics collector
	collector := metrics.NewCollector(&metrics.CollectorConfig{
		ServiceName:    serviceName,
		RecordInterval: 10 * time.Second,
		HistorySize:    100,
	})
	
	// Initialize and start the collector
	collector.Start()
	
	return collector
}

func setupSecurity(configManager *config.Manager, metricsCollector *metrics.Collector) *security.Monitor {
	// Create security components
	riskAnalyzer := security.NewRiskAnalyzer(configManager)
	eventProcessor := security.NewEventProcessor(metricsCollector)
	alertManager := security.NewAlertManager()
	
	// Create security monitor
	monitor := security.NewMonitor(riskAnalyzer, eventProcessor, alertManager)
	
	// Register detectors
	bruteForceDetector := security.NewBruteForceDetector(configManager)
	anomalyDetector := security.NewAnomalyDetector(metricsCollector)
	vulnerabilityDetector := security.NewVulnerabilityDetector()
	
	monitor.RegisterDetector(bruteForceDetector)
	monitor.RegisterDetector(anomalyDetector)
	monitor.RegisterDetector(vulnerabilityDetector)
	
	// Start monitoring
	monitor.Start()
	
	return monitor
}

func setupDiscovery(configManager *config.Manager, serviceName, serviceVersion string) discovery.Registry {
	// Create health checker with default timeout
	healthChecker := discovery.NewHealthChecker(5 * time.Second)
	
	// Create service registry
	registry := discovery.NewRegistry(healthChecker)
	
	// Set registry reference in health checker
	healthChecker.SetRegistry(registry)
	
	// Read service host/port from config
	serviceHost, _ := configManager.GetString("service.host")
	servicePort, _ := configManager.GetInt("service.port")
	
	if serviceHost == "" {
		serviceHost = "localhost"
	}
	
	if servicePort == 0 {
		servicePort = 8080 // Default port
	}
	
	// Register this service
	serviceAddress := fmt.Sprintf("%s:%d", serviceHost, servicePort)
	serviceID := fmt.Sprintf("%s-%s", serviceName, serviceVersion)
	
	instance := &discovery.ServiceInstance{
		ID:      serviceID,
		Name:    serviceName,
		Version: serviceVersion,
		Address: serviceAddress,
		Status:  discovery.StatusStarting,
		Tags:    []string{"api", "core"},
		Weight:  100,
	}
	
	options := &discovery.RegistrationOptions{
		TTL:                 discovery.DefaultTTL,
		AutoRenew:           true,
		HealthCheckInterval: 15 * time.Second,
		InitialStatus:       discovery.StatusStarting,
	}
	
	if err := registry.Register(instance, options); err != nil {
		log.Printf("Failed to register service: %v", err)
		return nil
	}
	
	// Update status to UP after initialization
	registry.UpdateStatus(serviceID, discovery.StatusUp)
	
	return registry
}

func setupServer(port int, metricsCollector *metrics.Collector, securityMonitor *security.Monitor) *http.Server {
	// Create router
	mux := http.NewServeMux()
	
	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"UP"}`))
	})
	
	// Add metrics endpoint
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement metrics reporting
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"metrics":"enabled"}`))
	})
	
	// Create security middleware
	securityMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create security event
			event := security.Event{
				Type:        security.EventTypeRequest,
				Timestamp:   time.Now(),
				ClientIP:    r.RemoteAddr,
				RequestPath: r.URL.Path,
				// Add more context from the request if needed
			}
			
			// Process security event (non-blocking)
			go securityMonitor.ProcessEvent(r.Context(), event)
			
			// Continue with the request
			next.ServeHTTP(w, r)
		})
	}
	
	// Create metrics middleware
	metricsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()
			
			// Wrap response writer to capture status code
			wrapper := newResponseWriter(w)
			
			// Process the request
			next.ServeHTTP(wrapper, r)
			
			// Record metrics
			duration := time.Since(startTime)
			metricsCollector.RecordMetric("http.request.duration", r.URL.Path, float64(duration.Milliseconds()), map[string]string{
				"method": r.Method,
				"status": fmt.Sprintf("%d", wrapper.status),
			})
		})
	}
	
	// Create server with middlewares
	var handler http.Handler = mux
	handler = securityMiddleware(handler)
	handler = metricsMiddleware(handler)
	
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	return server
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	status int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
