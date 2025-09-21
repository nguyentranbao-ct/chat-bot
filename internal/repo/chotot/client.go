package chotot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ChototAd represents the raw response from Chotot API
type ChototAd struct {
	AdID         int                 `json:"ad_id"`
	ListID       int                 `json:"list_id"`
	AccountID    int                 `json:"account_id"`
	AccountOID   string              `json:"account_oid"`
	Subject      string              `json:"subject"`
	Title        string              `json:"title"`
	Category     int                 `json:"category"`
	BigCate      int                 `json:"bigCate"`
	Price        int                 `json:"price"`
	PriceString  string              `json:"price_string"`
	Region       int                 `json:"region"`
	RegionName   string              `json:"region_name"`
	AreaV2       string              `json:"area_v2"`
	AreaName     string              `json:"area_name"`
	Date         string              `json:"date"`
	Images       []string            `json:"images"`
	Params       []ChototAdParam     `json:"params"`
}

type ChototAdParam struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type ChototResponse struct {
	Ads   []ChototAd `json:"ads"`
	Total int        `json:"total"`
}

type Client interface {
	GetUserAds(ctx context.Context, accountOID string, limit, page int) (*ChototResponse, error)
}

type client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient() Client {
	return &client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://gateway.chotot.org/v1/public/theia",
	}
}

func (c *client) GetUserAds(ctx context.Context, accountOID string, limit, page int) (*ChototResponse, error) {
	if limit <= 0 {
		limit = 9
	}
	if page <= 0 {
		page = 1
	}

	url := fmt.Sprintf("%s/%s?limit=%d&page=%d", c.baseURL, accountOID, limit, page)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var chototResp ChototResponse
	if err := json.NewDecoder(resp.Body).Decode(&chototResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chototResp, nil
}