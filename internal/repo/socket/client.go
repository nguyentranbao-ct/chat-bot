package socket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type Event struct {
	ProjectID string      `json:"project_id"`
	UserKey   string      `json:"user_key"`
	Platform  string      `json:"platform,omitempty"`
	Name      string      `json:"name"`
	Data      interface{} `json:"data"`
}

type SendEventsRequest struct {
	Events []Event `json:"events"`
}

type SendEventsResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func NewClient(conf *config.Config) *Client {
	return &Client{
		baseURL: conf.Socket.BaseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) SendEvents(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return nil
	}

	reqBody := SendEventsRequest{Events: events}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/events", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp SendEventsResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil && errorResp.Error != "" {
			return fmt.Errorf("socket server error: %s", errorResp.Error)
		}
		return fmt.Errorf("socket server returned status %d", resp.StatusCode)
	}

	var response SendEventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("socket server returned success=false: %s", response.Error)
	}

	log.Debugw(ctx, "sent events to socket server", "event_count", len(events))
	return nil
}

// BroadcastMessage sends a message_received event to a specific channel
// For now, this uses channel-based rooms, but should be updated to send to specific users
func (c *Client) BroadcastMessage(ctx context.Context, projectID, channelID string, message *models.ChatMessage) error {
	// Use channel ID as the user_key for channel-based rooms
	// In a production system, you'd want to get the list of channel members
	// and send individual events to each user using BroadcastMessageToUsers
	event := Event{
		ProjectID: projectID,
		UserKey:   fmt.Sprintf("channel_%s", channelID),
		Platform:  "web",
		Name:      "message_received",
		Data:      message,
	}

	return c.SendEvents(ctx, []Event{event})
}

// BroadcastMessageSent sends a message_sent confirmation to the sender
func (c *Client) BroadcastMessageSent(ctx context.Context, projectID, userID string, message *models.ChatMessage) error {
	event := Event{
		ProjectID: projectID,
		UserKey:   userID,
		Platform:  "web",
		Name:      "message_sent",
		Data:      message,
	}

	return c.SendEvents(ctx, []Event{event})
}

// BroadcastTyping sends typing indicator to channel members
func (c *Client) BroadcastTyping(ctx context.Context, projectID, channelID, userID string, isTyping bool) error {
	eventName := "user_typing_stop"
	if isTyping {
		eventName = "user_typing_start"
	}

	event := Event{
		ProjectID: projectID,
		UserKey:   fmt.Sprintf("channel_%s", channelID),
		Platform:  "web",
		Name:      eventName,
		Data: map[string]interface{}{
			"user_id":    userID,
			"channel_id": channelID,
			"is_typing":  isTyping,
		},
	}

	return c.SendEvents(ctx, []Event{event})
}

// BroadcastUserJoined notifies channel members when a user joins
func (c *Client) BroadcastUserJoined(ctx context.Context, projectID, channelID, userID string) error {
	event := Event{
		ProjectID: projectID,
		UserKey:   fmt.Sprintf("channel_%s", channelID),
		Platform:  "web",
		Name:      "user_joined",
		Data: map[string]interface{}{
			"user_id":    userID,
			"channel_id": channelID,
		},
	}

	return c.SendEvents(ctx, []Event{event})
}

// BroadcastUserLeft notifies channel members when a user leaves
func (c *Client) BroadcastUserLeft(ctx context.Context, projectID, channelID, userID string) error {
	event := Event{
		ProjectID: projectID,
		UserKey:   fmt.Sprintf("channel_%s", channelID),
		Platform:  "web",
		Name:      "user_left",
		Data: map[string]interface{}{
			"user_id":    userID,
			"channel_id": channelID,
		},
	}

	return c.SendEvents(ctx, []Event{event})
}

// BroadcastMessageToUsers sends a message_received event to a list of specific users
func (c *Client) BroadcastMessageToUsers(ctx context.Context, projectID string, userIDs []string, message *models.ChatMessage) error {
	if len(userIDs) == 0 {
		return nil
	}

	events := make([]Event, 0, len(userIDs))
	for _, userID := range userIDs {
		event := Event{
			ProjectID: projectID,
			UserKey:   userID, // Send directly to each user
			Platform:  "web",
			Name:      "message_received",
			Data:      message,
		}
		events = append(events, event)
	}

	return c.SendEvents(ctx, events)
}