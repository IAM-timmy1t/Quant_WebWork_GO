// server.go - HTTP server for metrics

package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/IAM-timmy1t/Quant_WebWork_GO/internal/core/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Server handles the serving of metrics via HTTP
type Server struct {
	config     config.MetricsConfig
	collector  *Collector
	logger     *zap.SugaredLogger
	httpServer *http.Server
}

// NewServer creates a new metrics server
func NewServer(config config.MetricsConfig, collector *Collector, logger *zap.SugaredLogger) *Server {
	return &Server{
		config:    config,
		collector: collector,
		logger:    logger,
	}
}

// Start starts the metrics server
func (s *Server) Start() error {
	if !s.config.Enabled {
		s.logger.Info("Metrics server is disabled")
		return nil
	}

	// Set up router with metrics endpoint
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// Add health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Configure and start HTTP server
	addr := fmt.Sprintf(":%d", 9090) // Default Prometheus port
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	s.logger.Infow("Starting metrics server", "address", addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// Stop stops the metrics server
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	s.logger.Info("Stopping metrics server")
	return s.httpServer.Shutdown(ctx)
}
