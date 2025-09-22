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
	UserID   string      `json:"user_id"`
	Platform string      `json:"platform,omitempty"`
	Name     string      `json:"name"`
	Data     interface{} `json:"data"`
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

type BroadcastMessageArgs struct {
	ChannelID string
	Message   *models.ChatMessage
}

// BroadcastMessage sends a message_received event to a specific channel
// This method is deprecated and should not be used - use BroadcastMessageToUsers instead
func (c *Client) BroadcastMessage(ctx context.Context, args BroadcastMessageArgs) error {
	// This method is deprecated - channel-based broadcasting should not be used
	// Instead, get channel members and use BroadcastMessageToUsers
	return fmt.Errorf("BroadcastMessage is deprecated - use BroadcastMessageToUsers instead")
}

type BroadcastMessageSentArgs struct {
	UserID  string
	Message *models.ChatMessage
}

// BroadcastMessageSent sends a message_sent confirmation to the sender
func (c *Client) BroadcastMessageSent(ctx context.Context, args BroadcastMessageSentArgs) error {
	event := Event{
		UserID:   args.UserID,
		Platform: "web",
		Name:     "message_sent",
		Data:     args.Message,
	}

	return c.SendEvents(ctx, []Event{event})
}

type BroadcastTypingArgs struct {
	ChannelID string
	UserID    string
	IsTyping  bool
}

// BroadcastTyping sends typing indicator to channel members
// This method is deprecated and should not be used - use BroadcastTypingToUsers instead
func (c *Client) BroadcastTyping(ctx context.Context, args BroadcastTypingArgs) error {
	// This method is deprecated - channel-based broadcasting should not be used
	// Instead, get channel members and use BroadcastTypingToUsers
	return fmt.Errorf("BroadcastTyping is deprecated - use BroadcastTypingToUsers instead")
}

type BroadcastUserJoinedArgs struct {
	ChannelID string
	UserID    string
}

// BroadcastUserJoined notifies channel members when a user joins
// This method is deprecated and should not be used - use BroadcastUserJoinedToUsers instead
func (c *Client) BroadcastUserJoined(ctx context.Context, args BroadcastUserJoinedArgs) error {
	// This method is deprecated - channel-based broadcasting should not be used
	// Instead, get channel members and use BroadcastUserJoinedToUsers
	return fmt.Errorf("BroadcastUserJoined is deprecated - use BroadcastUserJoinedToUsers instead")
}

type BroadcastUserLeftArgs struct {
	ChannelID string
	UserID    string
}

// BroadcastUserLeft notifies channel members when a user leaves
// This method is deprecated and should not be used - use BroadcastUserLeftToUsers instead
func (c *Client) BroadcastUserLeft(ctx context.Context, args BroadcastUserLeftArgs) error {
	// This method is deprecated - channel-based broadcasting should not be used
	// Instead, get channel members and use BroadcastUserLeftToUsers
	return fmt.Errorf("BroadcastUserLeft is deprecated - use BroadcastUserLeftToUsers instead")
}

type BroadcastMessageToUsersArgs struct {
	UserIDs []string
	Message *models.ChatMessage
}

// BroadcastMessageToUsers sends a message_received event to a list of specific users
func (c *Client) BroadcastMessageToUsers(ctx context.Context, args BroadcastMessageToUsersArgs) error {
	if len(args.UserIDs) == 0 {
		return nil
	}

	events := make([]Event, 0, len(args.UserIDs))
	for _, userID := range args.UserIDs {
		event := Event{
			UserID:   userID,
			Platform: "web",
			Name:     "message_received",
			Data:     args.Message,
		}
		events = append(events, event)
	}

	return c.SendEvents(ctx, events)
}

type BroadcastTypingToUsersArgs struct {
	UserIDs      []string
	ChannelID    string
	TypingUserID string
	IsTyping     bool
}

// BroadcastTypingToUsers sends typing indicator to specific users
func (c *Client) BroadcastTypingToUsers(ctx context.Context, args BroadcastTypingToUsersArgs) error {
	if len(args.UserIDs) == 0 {
		return nil
	}

	eventName := "user_typing_stop"
	if args.IsTyping {
		eventName = "user_typing_start"
	}

	events := make([]Event, 0, len(args.UserIDs))
	for _, userID := range args.UserIDs {
		// Don't send typing indicator to the user who is typing
		if userID == args.TypingUserID {
			continue
		}

		event := Event{
			UserID:   userID,
			Platform: "web",
			Name:     eventName,
			Data: map[string]interface{}{
				"user_id":    args.TypingUserID,
				"channel_id": args.ChannelID,
				"is_typing":  args.IsTyping,
			},
		}
		events = append(events, event)
	}

	return c.SendEvents(ctx, events)
}

type BroadcastUserJoinedToUsersArgs struct {
	UserIDs      []string
	ChannelID    string
	JoinedUserID string
}

// BroadcastUserJoinedToUsers notifies specific users when a user joins
func (c *Client) BroadcastUserJoinedToUsers(ctx context.Context, args BroadcastUserJoinedToUsersArgs) error {
	if len(args.UserIDs) == 0 {
		return nil
	}

	events := make([]Event, 0, len(args.UserIDs))
	for _, userID := range args.UserIDs {
		event := Event{
			UserID:   userID,
			Platform: "web",
			Name:     "user_joined",
			Data: map[string]interface{}{
				"user_id":    args.JoinedUserID,
				"channel_id": args.ChannelID,
			},
		}
		events = append(events, event)
	}

	return c.SendEvents(ctx, events)
}

type BroadcastUserLeftToUsersArgs struct {
	UserIDs    []string
	ChannelID  string
	LeftUserID string
}

// BroadcastUserLeftToUsers notifies specific users when a user leaves
func (c *Client) BroadcastUserLeftToUsers(ctx context.Context, args BroadcastUserLeftToUsersArgs) error {
	if len(args.UserIDs) == 0 {
		return nil
	}

	events := make([]Event, 0, len(args.UserIDs))
	for _, userID := range args.UserIDs {
		event := Event{
			UserID:   userID,
			Platform: "web",
			Name:     "user_left",
			Data: map[string]interface{}{
				"user_id":    args.LeftUserID,
				"channel_id": args.ChannelID,
			},
		}
		events = append(events, event)
	}

	return c.SendEvents(ctx, events)
}
