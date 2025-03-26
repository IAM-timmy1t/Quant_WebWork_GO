// manager.go - IP masking implementation

package ipmasking

import (
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Manager manages IP masking
type Manager struct {
	enabled             bool
	rotationInterval    time.Duration
	preserveGeolocation bool
	dnsPrivacyEnabled   bool
	mappings            map[string]string
	logger              *zap.SugaredLogger
	mutex               sync.RWMutex
	stopRotation        chan struct{}
}

// NewManager creates a new IP masking manager
func NewManager(logger *zap.SugaredLogger) *Manager {
	if logger == nil {
		// Create a no-op logger if none provided
		noop := zap.NewNop()
		logger = noop.Sugar()
	}

	return &Manager{
		enabled:             false,
		rotationInterval:    1 * time.Hour,
		preserveGeolocation: true,
		dnsPrivacyEnabled:   true,
		mappings:            make(map[string]string),
		logger:              logger,
		stopRotation:        make(chan struct{}),
	}
}

// NewManagerWithOptions creates a new IP masking manager with options
func NewManagerWithOptions(options *MaskingOptions, logger *zap.SugaredLogger) *Manager {
	if logger == nil {
		// Create a no-op logger if none provided
		noop := zap.NewNop()
		logger = noop.Sugar()
	}

	if options == nil {
		options = DefaultMaskingOptions()
	}

	manager := &Manager{
		enabled:             false,
		rotationInterval:    options.RotationInterval,
		preserveGeolocation: options.PreserveGeolocation,
		dnsPrivacyEnabled:   options.DNSPrivacyEnabled,
		mappings:            make(map[string]string),
		logger:              logger,
		stopRotation:        make(chan struct{}),
	}

	// Auto-start if configured
	if options.AutoStart {
		if err := manager.Start(); err != nil {
			logger.Warnw("Failed to auto-start IP masking", "error", err)
		}
	}

	return manager
}

// Start starts IP masking
func (m *Manager) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.enabled {
		return nil // Already running
	}

	m.enabled = true
	m.logger.Info("IP masking started")

	// Start the rotation routine
	go m.rotateIPs()

	return nil
}

// Stop stops IP masking
func (m *Manager) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.enabled {
		return nil // Already stopped
	}

	m.enabled = false
	close(m.stopRotation)

	// Create a new channel for future starts
	m.stopRotation = make(chan struct{})

	m.logger.Info("IP masking stopped")
	return nil
}

// IsEnabled returns whether IP masking is enabled
func (m *Manager) IsEnabled() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.enabled
}

// GetMaskedIP returns a masked IP for the specified original IP
func (m *Manager) GetMaskedIP(originalIP net.IP) net.IP {
	if !m.IsEnabled() {
		return originalIP
	}

	originalIPStr := originalIP.String()

	m.mutex.RLock()
	maskedIPStr, ok := m.mappings[originalIPStr]
	m.mutex.RUnlock()

	if ok {
		return net.ParseIP(maskedIPStr)
	}

	// Generate a new masked IP
	maskedIP := m.generateMaskedIP(originalIP)

	m.mutex.Lock()
	m.mappings[originalIPStr] = maskedIP.String()
	m.mutex.Unlock()

	return maskedIP
}

// SetRotationInterval sets the IP rotation interval
func (m *Manager) SetRotationInterval(interval time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.rotationInterval = interval
	m.logger.Infow("Updated rotation interval", "interval", interval)
}

// SetPreserveGeolocation sets whether to preserve geolocation
func (m *Manager) SetPreserveGeolocation(preserve bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.preserveGeolocation = preserve
	m.logger.Infow("Updated preserve geolocation setting", "preserve", preserve)
}

// SetDNSPrivacyEnabled sets whether DNS privacy is enabled
func (m *Manager) SetDNSPrivacyEnabled(enabled bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.dnsPrivacyEnabled = enabled
	m.logger.Infow("Updated DNS privacy setting", "enabled", enabled)
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return map[string]interface{}{
		"enabled":             m.enabled,
		"rotationInterval":    m.rotationInterval.String(),
		"preserveGeolocation": m.preserveGeolocation,
		"dnsPrivacyEnabled":   m.dnsPrivacyEnabled,
		"activeIPMappings":    len(m.mappings),
	}
}

// rotateIPs periodically rotates IPs
func (m *Manager) rotateIPs() {
	ticker := time.NewTicker(m.rotationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopRotation:
			return
		case <-ticker.C:
			if !m.IsEnabled() {
				return
			}

			m.mutex.Lock()
			// Save the count before clearing for logging
			oldCount := len(m.mappings)
			// Clear all mappings to force new ones to be generated
			m.mappings = make(map[string]string)
			m.mutex.Unlock()

			m.logger.Infow("Rotated IP mappings", "previousCount", oldCount)
		}
	}
}

// generateMaskedIP generates a masked IP for the specified original IP
func (m *Manager) generateMaskedIP(originalIP net.IP) net.IP {
	// This is a simplified implementation
	// In a production environment, this would use more sophisticated techniques

	// For IPv4
	if ipv4 := originalIP.To4(); ipv4 != nil {
		// Keep the network portion and mask the host portion
		if m.preserveGeolocation {
			// For geolocation preservation, keep first two octets
			// This is a simplified approach, production should use actual geo-mapping
			return net.IPv4(ipv4[0], ipv4[1], byte(time.Now().Nanosecond()%256), byte(time.Now().UnixNano()%256))
		}
		// Otherwise completely randomize the IP
		return net.IPv4(
			byte(time.Now().Second()%256),
			byte(time.Now().Nanosecond()%256),
			byte(time.Now().Unix()%256),
			byte(time.Now().UnixNano()%256),
		)
	}

	// For IPv6
	// Create a masked IPv6 address
	// In production, this should be more sophisticated
	if m.preserveGeolocation {
		// Keep first 6 bytes for geo-preservation (simplified)
		maskedIP := make(net.IP, len(originalIP))
		copy(maskedIP, originalIP)
		// Randomize the rest
		for i := 6; i < len(maskedIP); i++ {
			maskedIP[i] = byte(time.Now().UnixNano()%256 + int64(i))
		}
		return maskedIP
	}

	// Create a completely randomized IPv6
	maskedIP := make(net.IP, 16)
	for i := range maskedIP {
		maskedIP[i] = byte(time.Now().UnixNano()%256 + int64(i))
	}
	// Ensure it's a valid IPv6
	maskedIP[0] = 0xfd // Use fd00::/8 for unique local addresses
	return maskedIP
}

// GetMappingCount returns the number of active IP mappings
func (m *Manager) GetMappingCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return len(m.mappings)
}

// ClearMappings clears all IP mappings
func (m *Manager) ClearMappings() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	oldCount := len(m.mappings)
	m.mappings = make(map[string]string)

	m.logger.Infow("Cleared IP mappings", "count", oldCount)
}

// GetOriginalIPForMasked attempts to reverse lookup a masked IP to find the original
// This is mostly for diagnostic purposes
func (m *Manager) GetOriginalIPForMasked(maskedIP net.IP) (net.IP, bool) {
	maskedIPStr := maskedIP.String()

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for originalIPStr, currentMaskedIPStr := range m.mappings {
		if currentMaskedIPStr == maskedIPStr {
			return net.ParseIP(originalIPStr), true
		}
	}

	return nil, false
}
