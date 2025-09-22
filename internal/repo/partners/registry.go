package partners

import (
	"context"
	"fmt"
	"strings"
	"sync"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
)

// PartnerRegistry manages partner instances and provides partner detection
type PartnerRegistry struct {
	partners map[PartnerType]Partner
	mu       sync.RWMutex
}

// NewPartnerRegistry creates a new partner registry
func NewPartnerRegistry() *PartnerRegistry {
	return &PartnerRegistry{
		partners: make(map[PartnerType]Partner),
	}
}

// RegisterPartner registers a partner instance
func (r *PartnerRegistry) RegisterPartner(partner Partner) error {
	if partner == nil {
		return fmt.Errorf("partner cannot be nil")
	}

	partnerType := partner.GetPartnerType()
	if partnerType == "" {
		return fmt.Errorf("partner type cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.partners[partnerType]; exists {
		return fmt.Errorf("partner type %s already registered", partnerType)
	}

	r.partners[partnerType] = partner
	return nil
}

// GetPartner retrieves a partner by type
func (r *PartnerRegistry) GetPartner(partnerType PartnerType) (Partner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	partner, exists := r.partners[partnerType]
	if !exists {
		return nil, fmt.Errorf("partner type %s not found", partnerType)
	}

	return partner, nil
}

// GetPartnerByName retrieves a partner by string name (case-insensitive)
func (r *PartnerRegistry) GetPartnerByName(name string) (Partner, error) {
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	return r.GetPartner(PartnerType(normalizedName))
}

// ListPartners returns all registered partner types
func (r *PartnerRegistry) ListPartners() []PartnerType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	partners := make([]PartnerType, 0, len(r.partners))
	for partnerType := range r.partners {
		partners = append(partners, partnerType)
	}
	return partners
}

// HealthCheckAll performs health checks on all registered partners
func (r *PartnerRegistry) HealthCheckAll(ctx context.Context) map[PartnerType]error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[PartnerType]error)
	for partnerType, partner := range r.partners {
		err := partner.HealthCheck(ctx)
		results[partnerType] = err

		if err != nil {
			log.Errorw(ctx, "Partner health check failed",
				"partner_type", partnerType,
				"error", err)
		} else {
			log.Debugw(ctx, "Partner health check passed",
				"partner_type", partnerType)
		}
	}
	return results
}

// GetCapabilities returns capabilities for all registered partners
func (r *PartnerRegistry) GetCapabilities() map[PartnerType]PartnerCapabilities {
	r.mu.RLock()
	defer r.mu.RUnlock()

	capabilities := make(map[PartnerType]PartnerCapabilities)
	for partnerType, partner := range r.partners {
		capabilities[partnerType] = partner.GetCapabilities()
	}
	return capabilities
}

// ValidatePartnerName checks if a partner name is valid and registered
func (r *PartnerRegistry) ValidatePartnerName(name string) error {
	if name == "" {
		return fmt.Errorf("partner name cannot be empty")
	}

	normalizedName := strings.ToLower(strings.TrimSpace(name))
	_, err := r.GetPartner(PartnerType(normalizedName))
	return err
}
