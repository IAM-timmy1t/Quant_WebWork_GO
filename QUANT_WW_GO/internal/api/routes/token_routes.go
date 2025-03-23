// token_routes.go - Routes for token analysis API

package routes

import (
	"net/http"

	"github.com/quant-webworks/go/internal/api/handlers"
	"github.com/quant-webworks/go/internal/security/risk"
)

// RegisterTokenRoutes registers all token-related routes
func RegisterTokenRoutes(router *http.ServeMux, tokenAnalyzer *risk.TokenAnalyzer) {
	// Create handlers
	tokenHandler := handlers.NewTokenAnalysisHandler(tokenAnalyzer)
	
	// Register routes
	router.HandleFunc("/api/security/token-analysis", tokenHandler.HandleTokenAnalysis)
	
	// For backwards compatibility
	router.HandleFunc("/api/v1/security/token-analysis", tokenHandler.HandleTokenAnalysis)
}
