package chotot

import (
	"context"
	"fmt"

	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/list_products"
)

type ProductService struct {
	client Client
}

func NewProductService(client Client) *ProductService {
	return &ProductService{
		client: client,
	}
}

func (s *ProductService) ListUserProducts(ctx context.Context, accountOID string, limit, page int) ([]list_products.Product, int, error) {
	response, err := s.client.GetUserAds(ctx, accountOID, limit, page)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user ads: %w", err)
	}

	products := make([]list_products.Product, 0, len(response.Ads))
	for _, ad := range response.Ads {
		products = append(products, s.mapToProduct(ad))
	}

	return products, response.Total, nil
}

func (s *ProductService) mapToProduct(ad ChototAd) list_products.Product {
	return list_products.Product{
		ID:          fmt.Sprintf("%d", ad.ListID),
		Name:        ad.Subject,
		Category:    s.getCategoryName(ad.Category),
		Price:       ad.Price,
		PriceString: ad.PriceString,
		Images:      ad.Images,
		Source:      fmt.Sprintf("chotot://%d", ad.ListID),
	}
}

// getCategoryName maps category ID to string
// This is a simple implementation - in a real system you might want to fetch this from an API
func (s *ProductService) getCategoryName(categoryID int) string {
	categoryMap := map[int]string{
		5000: "Electronics",
		2000: "Vehicles",
		1000: "Real Estate",
		9000: "Home & Garden",
		7000: "Fashion",
		// Add more categories as needed
	}

	if name, exists := categoryMap[categoryID]; exists {
		return name
	}
	return fmt.Sprintf("Category_%d", categoryID)
}