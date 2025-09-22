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
	client *Client
}

func NewBroadcaster(client *Client) *Broadcaster {
	return &Broadcaster{
		client: client,
	}
}

// BroadcastMessage broadcasts a new message to all room members
func (b *Broadcaster) BroadcastMessage(roomID string, message *models.ChatMessage) {
	ctx := context.Background()
	args := BroadcastMessageArgs{
		RoomID:  roomID,
		Message: message,
	}
	if err := b.client.BroadcastMessage(ctx, args); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to broadcast message to room %s: %v\n", roomID, err)
	}
}

// BroadcastMessageSent broadcasts message sent confirmation to the sender
func (b *Broadcaster) BroadcastMessageSent(userID string, message *models.ChatMessage) {
	ctx := context.Background()
	args := BroadcastMessageSentArgs{
		UserID:  userID,
		Message: message,
	}
	if err := b.client.BroadcastMessageSent(ctx, args); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to broadcast message sent to user %s: %v\n", userID, err)
	}
}

// BroadcastTyping broadcasts typing indicator to room members
func (b *Broadcaster) BroadcastTyping(roomID, userID string, isTyping bool) {
	ctx := context.Background()
	args := BroadcastTypingArgs{
		RoomID:   roomID,
		UserID:   userID,
		IsTyping: isTyping,
	}
	if err := b.client.BroadcastTyping(ctx, args); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to broadcast typing status to room %s: %v\n", roomID, err)
	}
}

// BroadcastMessageToUsers broadcasts a message to specific users
func (b *Broadcaster) BroadcastMessageToUsers(userIDs []string, message *models.ChatMessage) {
	ctx := context.Background()
	args := BroadcastMessageToUsersArgs{
		UserIDs: userIDs,
		Message: message,
	}
	if err := b.client.BroadcastMessageToUsers(ctx, args); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to broadcast message to users %v: %v\n", userIDs, err)
	}
}

// BroadcastTypingToUsers broadcasts typing indicator to specific users
func (b *Broadcaster) BroadcastTypingToUsers(userIDs []string, roomID, userID string, isTyping bool) {
	ctx := context.Background()
	args := BroadcastTypingToUsersArgs{
		UserIDs:      userIDs,
		RoomID:       roomID,
		TypingUserID: userID,
		IsTyping:     isTyping,
	}
	if err := b.client.BroadcastTypingToUsers(ctx, args); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to broadcast typing status to users %v: %v\n", userIDs, err)
	}
}
