package socket

import (
	"context"
	"fmt"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/usecase"
)

// Ensure Broadcaster implements SocketBroadcaster interface
var _ usecase.SocketBroadcaster = (*Broadcaster)(nil)

type Broadcaster struct {
	client    *Client
	projectID string
}

func NewBroadcaster(client *Client, projectID string) *Broadcaster {
	return &Broadcaster{
		client:    client,
		projectID: projectID,
	}
}

// BroadcastMessage broadcasts a new message to all channel members
func (b *Broadcaster) BroadcastMessage(channelID string, message *models.ChatMessage) {
	ctx := context.Background()
	if err := b.client.BroadcastMessage(ctx, b.projectID, channelID, message); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to broadcast message to channel %s: %v\n", channelID, err)
	}
}

// BroadcastMessageSent broadcasts message sent confirmation to the sender
func (b *Broadcaster) BroadcastMessageSent(userID string, message *models.ChatMessage) {
	ctx := context.Background()
	if err := b.client.BroadcastMessageSent(ctx, b.projectID, userID, message); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to broadcast message sent to user %s: %v\n", userID, err)
	}
}

// BroadcastTyping broadcasts typing indicator to channel members
func (b *Broadcaster) BroadcastTyping(channelID, userID string, isTyping bool) {
	ctx := context.Background()
	if err := b.client.BroadcastTyping(ctx, b.projectID, channelID, userID, isTyping); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to broadcast typing status to channel %s: %v\n", channelID, err)
	}
}

// BroadcastMessageToUsers broadcasts a message to specific users
func (b *Broadcaster) BroadcastMessageToUsers(userIDs []string, message *models.ChatMessage) {
	ctx := context.Background()
	if err := b.client.BroadcastMessageToUsers(ctx, b.projectID, userIDs, message); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to broadcast message to users %v: %v\n", userIDs, err)
	}
}