// types.go - IP masking type definitions

package ipmasking

import (
	"net"
	"time"
)

// IPMasker defines the interface for IP masking functionality
type IPMasker interface {
	// Start starts IP masking
	Start() error

	// Stop stops IP masking
	Stop() error

	// IsEnabled returns whether IP masking is enabled
	IsEnabled() bool

	// GetMaskedIP returns a masked IP for the specified original IP
	GetMaskedIP(originalIP net.IP) net.IP

	// SetRotationInterval sets the IP rotation interval
	SetRotationInterval(interval time.Duration)

	// SetPreserveGeolocation sets whether to preserve geolocation
	SetPreserveGeolocation(preserve bool)

	// SetDNSPrivacyEnabled sets whether DNS privacy is enabled
	SetDNSPrivacyEnabled(enabled bool)

	// GetConfig returns the current configuration
	GetConfig() map[string]interface{}

	// GetMappingCount returns the number of active IP mappings
	GetMappingCount() int

	// ClearMappings clears all IP mappings
	ClearMappings()

	// GetOriginalIPForMasked attempts to reverse lookup a masked IP
	GetOriginalIPForMasked(maskedIP net.IP) (net.IP, bool)
}

// Ensure Manager implements IPMasker
var _ IPMasker = (*Manager)(nil)

// MaskingOptions defines options for IP masking
type MaskingOptions struct {
	// RotationInterval is how often to rotate IP mappings
	RotationInterval time.Duration

	// PreserveGeolocation determines whether to preserve geolocation info
	PreserveGeolocation bool

	// DNSPrivacyEnabled determines whether to apply privacy for DNS
	DNSPrivacyEnabled bool

	// AutoStart determines whether masking starts immediately on creation
	AutoStart bool
}

// DefaultMaskingOptions returns default masking options
func DefaultMaskingOptions() *MaskingOptions {
	return &MaskingOptions{
		RotationInterval:    1 * time.Hour,
		PreserveGeolocation: true,
		DNSPrivacyEnabled:   true,
		AutoStart:           false,
	}
}
