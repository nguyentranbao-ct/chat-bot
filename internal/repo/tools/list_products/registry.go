package list_products

import "sync"

type productServiceRegistry struct {
	services map[string]ProductService
	mu       sync.RWMutex
}

func NewProductServiceRegistry() ProductServiceRegistry {
	return &productServiceRegistry{
		services: make(map[string]ProductService),
	}
}

func (r *productServiceRegistry) RegisterService(linkType string, service ProductService) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[linkType] = service
}

func (r *productServiceRegistry) GetService(linkType string) (ProductService, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	service, exists := r.services[linkType]
	return service, exists
}