package rest

import (
	"encoding/json"
	"net/http"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	TOTPCode string `json:"totp_code,omitempty"`
}

type loginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type refreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// handleLogin processes user login requests
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// TODO: Implement user authentication against database
	// For now, we'll use a mock authentication
	if req.Username != "admin" || req.Password != "password" {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// For master admin, verify TOTP
	if req.Username == "admin" {
		if req.TOTPCode == "" {
			respondError(w, http.StatusBadRequest, "TOTP code required for admin login")
			return
		}

		// TODO: Get master secret from secure storage
		masterSecret := "MASTER_SECRET"
		if !h.totpManager.ValidateCode(masterSecret, req.TOTPCode) {
			if h.securityLock.AddViolation(r.RemoteAddr) {
				respondError(w, http.StatusTooManyRequests, "Too many failed attempts")
				return
			}
			respondError(w, http.StatusUnauthorized, "Invalid TOTP code")
			return
		}
	}

	// Generate JWT token
	token, err := h.authManager.GenerateToken("1", req.Username, "master")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// TODO: Generate and store refresh token
	refreshToken := "mock-refresh-token"

	resp := loginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresIn:    3600, // 1 hour
	}

	respondJSON(w, http.StatusOK, resp)
}

// handleRefreshToken processes token refresh requests
func (h *Handler) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	var req refreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// TODO: Validate refresh token against storage
	if req.RefreshToken != "mock-refresh-token" {
		respondError(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	// Generate new JWT token
	token, err := h.authManager.GenerateToken("1", "admin", "master")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	resp := loginResponse{
		Token:        token,
		RefreshToken: req.RefreshToken, // Reuse the same refresh token
		ExpiresIn:    3600,             // 1 hour
	}

	respondJSON(w, http.StatusOK, resp)
}

// handleGenerate2FA generates new 2FA credentials for master admin
func (h *Handler) handleGenerate2FA(w http.ResponseWriter, r *http.Request) {
	secret, err := h.totpManager.GenerateSecret()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate secret")
		return
	}

	key, err := h.totpManager.GenerateQRCode(secret)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate QR code")
		return
	}

	resp := map[string]string{
		"secret": secret,
		"qr_url": key.URL(),
	}

	respondJSON(w, http.StatusOK, resp)
}

// handleResetSecurity resets security violations for an IP
func (h *Handler) handleResetSecurity(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IP string `json:"ip"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// TODO: Implement reset logic in SecurityLock

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "security reset successful",
	})
}
