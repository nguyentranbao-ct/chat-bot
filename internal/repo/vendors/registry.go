package vendors

import (
	"context"
	"fmt"
	"strings"
	"sync"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
)

// VendorRegistry manages vendor instances and provides vendor detection
type VendorRegistry struct {
	vendors map[VendorType]Vendor
	mu      sync.RWMutex
}

// NewVendorRegistry creates a new vendor registry
func NewVendorRegistry() *VendorRegistry {
	return &VendorRegistry{
		vendors: make(map[VendorType]Vendor),
	}
}

// RegisterVendor registers a vendor instance
func (r *VendorRegistry) RegisterVendor(vendor Vendor) error {
	if vendor == nil {
		return fmt.Errorf("vendor cannot be nil")
	}

	vendorType := vendor.GetVendorType()
	if vendorType == "" {
		return fmt.Errorf("vendor type cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.vendors[vendorType]; exists {
		return fmt.Errorf("vendor type %s already registered", vendorType)
	}

	r.vendors[vendorType] = vendor
	return nil
}

// GetVendor retrieves a vendor by type
func (r *VendorRegistry) GetVendor(vendorType VendorType) (Vendor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	vendor, exists := r.vendors[vendorType]
	if !exists {
		return nil, fmt.Errorf("vendor type %s not found", vendorType)
	}

	return vendor, nil
}

// GetVendorByName retrieves a vendor by string name (case-insensitive)
func (r *VendorRegistry) GetVendorByName(name string) (Vendor, error) {
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	return r.GetVendor(VendorType(normalizedName))
}

// ListVendors returns all registered vendor types
func (r *VendorRegistry) ListVendors() []VendorType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	vendors := make([]VendorType, 0, len(r.vendors))
	for vendorType := range r.vendors {
		vendors = append(vendors, vendorType)
	}
	return vendors
}

// detectVendorByPattern detects vendor based on channel ID patterns
func (r *VendorRegistry) detectVendorByPattern(channelID string) VendorType {
	// Facebook Messenger patterns
	if strings.Contains(channelID, "facebook") ||
		strings.Contains(channelID, "messenger") ||
		strings.HasPrefix(channelID, "fb_") {
		return VendorTypeFacebook
	}

	// Add more patterns as needed for other vendors
	// Telegram: strings.HasPrefix(channelID, "tg_")
	// WhatsApp: strings.HasPrefix(channelID, "wa_")

	// Default case - return empty string to indicate no pattern match
	return ""
}

// HealthCheckAll performs health checks on all registered vendors
func (r *VendorRegistry) HealthCheckAll(ctx context.Context) map[VendorType]error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[VendorType]error)
	for vendorType, vendor := range r.vendors {
		err := vendor.HealthCheck(ctx)
		results[vendorType] = err

		if err != nil {
			log.Errorw(ctx, "Vendor health check failed",
				"vendor_type", vendorType,
				"error", err)
		} else {
			log.Debugw(ctx, "Vendor health check passed",
				"vendor_type", vendorType)
		}
	}
	return results
}

// GetCapabilities returns capabilities for all registered vendors
func (r *VendorRegistry) GetCapabilities() map[VendorType]VendorCapabilities {
	r.mu.RLock()
	defer r.mu.RUnlock()

	capabilities := make(map[VendorType]VendorCapabilities)
	for vendorType, vendor := range r.vendors {
		capabilities[vendorType] = vendor.GetCapabilities()
	}
	return capabilities
}

// ValidateVendorName checks if a vendor name is valid and registered
func (r *VendorRegistry) ValidateVendorName(name string) error {
	if name == "" {
		return fmt.Errorf("vendor name cannot be empty")
	}

	normalizedName := strings.ToLower(strings.TrimSpace(name))
	_, err := r.GetVendor(VendorType(normalizedName))
	return err
}
