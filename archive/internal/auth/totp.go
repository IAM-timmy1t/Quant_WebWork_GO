package auth

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTPConfig holds the configuration for TOTP generation
type TOTPConfig struct {
	Issuer      string
	AccountName string
	SecretSize  int
}

// TOTPManager handles TOTP operations
type TOTPManager struct {
	config TOTPConfig
}

// NewTOTPManager creates a new TOTP manager
func NewTOTPManager(config TOTPConfig) *TOTPManager {
	if config.SecretSize == 0 {
		config.SecretSize = 20 // Default secret size
	}
	return &TOTPManager{config: config}
}

// GenerateSecret creates a new TOTP secret
func (m *TOTPManager) GenerateSecret() (string, error) {
	bytes := make([]byte, m.config.SecretSize)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %v", err)
	}
	
	return base32.StdEncoding.EncodeToString(bytes), nil
}

// GenerateQRCode generates a QR code for TOTP setup
func (m *TOTPManager) GenerateQRCode(secret string) (*otp.Key, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      m.config.Issuer,
		AccountName: m.config.AccountName,
		Secret:      []byte(secret),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP key: %v", err)
	}
	
	return key, nil
}

// ValidateCode validates a TOTP code
func (m *TOTPManager) ValidateCode(secret string, code string) bool {
	return totp.Validate(code, secret)
}

// GenerateCode generates a new TOTP code (useful for testing)
func (m *TOTPManager) GenerateCode(secret string) (string, error) {
	return totp.GenerateCode(secret, time.Now())
}
