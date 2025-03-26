package security

import (
    "errors"
    "os"
    
    "go.uber.org/zap"
)

// EnvironmentType represents the type of environment
type EnvironmentType string

const (
    EnvDevelopment EnvironmentType = "development"
    EnvStaging     EnvironmentType = "staging" 
    EnvProduction  EnvironmentType = "production"
)

// SecurityConfig represents the security configuration
type SecurityConfig struct {
    Environment       EnvironmentType
    AuthRequired      bool
    AdminCredentials  bool
    TLSRequired       bool
    StrictFirewall    bool
    IPMaskingEnabled  bool
    RateLimitingLevel string // "off", "basic", "strict"
    AuditLoggingLevel string // "off", "basic", "verbose"
}

// GetEnvironmentType determines the environment type
func GetEnvironmentType() EnvironmentType {
    env := os.Getenv("QUANT_ENV")
    switch env {
    case "production":
        return EnvProduction
    case "staging":
        return EnvStaging
    default:
        return EnvDevelopment
    }
}

// IsLocalEnvironment checks if running in a local environment
func IsLocalEnvironment() bool {
    hostname, err := os.Hostname()
    if err != nil {
        return false
    }
    
    // Check if hostname is likely a local development machine
    if hostname == "localhost" || hostname == "127.0.0.1" {
        return true
    }
    
    // Could add additional checks for local Docker, etc.
    return false
}

// GetSecurityConfig returns security configuration based on environment
func GetSecurityConfig(logger *zap.SugaredLogger) SecurityConfig {
    envType := GetEnvironmentType()
    isLocal := IsLocalEnvironment()
    
    config := SecurityConfig{
        Environment: envType,
    }
    
    switch envType {
    case EnvProduction:
        // Production is secure by default
        config.AuthRequired = true
        config.AdminCredentials = true
        config.TLSRequired = true
        config.StrictFirewall = true
        config.IPMaskingEnabled = true
        config.RateLimitingLevel = "strict"
        config.AuditLoggingLevel = "verbose"
        
    case EnvStaging:
        // Staging is mostly secure but might allow some flexibility
        config.AuthRequired = true
        config.AdminCredentials = true
        config.TLSRequired = true
        config.StrictFirewall = false
        config.IPMaskingEnabled = true
        config.RateLimitingLevel = "basic"
        config.AuditLoggingLevel = "basic"
        
    default: // Development
        // Development prioritizes convenience, but warns about insecurity
        config.AuthRequired = false
        config.AdminCredentials = false
        config.TLSRequired = false
        config.StrictFirewall = false
        config.IPMaskingEnabled = false
        config.RateLimitingLevel = "off"
        config.AuditLoggingLevel = "verbose" // Log everything in dev
    }
    
    // Override for non-local production to enforce security
    if envType == EnvProduction && !isLocal {
        // Force security for non-local production
        if !config.AuthRequired || !config.AdminCredentials {
            logger.Warn("SECURITY RISK: Running in production without authentication!")
            logger.Warn("Forcing authentication for production environment")
            config.AuthRequired = true
            config.AdminCredentials = true
        }
        
        if !config.TLSRequired {
            logger.Warn("SECURITY RISK: Running in production without TLS!")
            logger.Warn("Forcing TLS for production environment")
            config.TLSRequired = true
        }
    }
    
    return config
}

// ValidateProductionSecurity validates security for production deployment
func ValidateProductionSecurity(config SecurityConfig) error {
    if config.Environment == EnvProduction {
        if !config.AuthRequired {
            return errors.New("authentication must be enabled in production environment")
        }
        
        if !config.TLSRequired {
            return errors.New("TLS must be enabled in production environment")
        }
        
        // Add other security validations as needed
    }
    
    return nil
}
