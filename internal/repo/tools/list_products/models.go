package list_products

import "context"

// Product represents our unified product model
type Product struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Price       int      `json:"price"`
	PriceString string   `json:"price_string"`
	Images      []string `json:"images"`
	Source      string   `json:"source"` // e.g., "chotot://1823180"
}

// ProductService interface for pluggable product services
type ProductService interface {
	ListUserProducts(ctx context.Context, userID string, limit, page int) ([]Product, int, error)
}

// ProductServiceRegistry interface for managing product services
type ProductServiceRegistry interface {
	RegisterService(linkType string, service ProductService)
	GetService(linkType string) (ProductService, bool)
}
