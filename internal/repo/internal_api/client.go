package internal_api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nguyentranbao-ct/chat-bot/internal/config"
)

type Client interface {
	SendMessage(ctx context.Context, req SendMessageRequest) error
}

type SendMessageRequest struct {
	ChannelID   string `json:"channel_id" validate:"required"`
	SenderID    string `json:"sender_id" validate:"required"`
	Content     string `json:"content" validate:"required"`
	SkipPartner bool   `json:"skip_partner,omitempty"`
}

type client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(cfg *config.Config) Client {
	return &client{
		httpClient: &http.Client{},
		baseURL:    fmt.Sprintf("http://%s/api/v1", cfg.Server.Addr),
	}
}

func (c *client) SendMessage(ctx context.Context, req SendMessageRequest) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/internal/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
