package chotot

import (
	"context"
	"fmt"

	"github.com/nguyentranbao-ct/chat-bot/internal/repo/tools/list_products"
	"github.com/spf13/cast"
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
		products = append(products, s.mapToProduct(ad.Info))
	}

	return products, response.Total, nil
}

func (s *ProductService) mapToProduct(ad ChototAd) list_products.Product {
	// Ensure images is never nil to avoid schema validation errors
	images := ad.Images
	if images == nil {
		images = []string{}
	}

	return list_products.Product{
		ID:          fmt.Sprintf("%d", ad.ListID),
		Name:        ad.Subject,
		Category:    cast.ToString(ad.Category),
		Price:       cast.ToInt(ad.Price),
		PriceString: ad.PriceString,
		Images:      images,
		Source:      fmt.Sprintf("chotot://%d", ad.ListID),
	}
}
