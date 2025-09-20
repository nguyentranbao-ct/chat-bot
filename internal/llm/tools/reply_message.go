package tools

import (
	"context"
	"fmt"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/nguyentranbao-ct/chat-bot/internal/client"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repository"
)

type ReplyMessageArgs struct {
	Message string `json:"message"`
}

type ReplyMessageTool struct {
	chatAPIClient client.ChatAPIClient
	activityRepo  repository.ChatActivityRepository
}

func NewReplyMessageTool(
	chatAPIClient client.ChatAPIClient,
	activityRepo repository.ChatActivityRepository,
) *ReplyMessageTool {
	return &ReplyMessageTool{
		chatAPIClient: chatAPIClient,
		activityRepo:  activityRepo,
	}
}

func (t *ReplyMessageTool) Execute(ctx context.Context, args ReplyMessageArgs, session SessionContext) error {
	message := &models.OutgoingMessage{
		ChannelID: session.GetChannelID(),
		SenderID:  session.GetSenderID(),
		Message:   args.Message,
	}

	if err := t.chatAPIClient.SendMessage(ctx, message); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	if err := t.logActivity(ctx, args, session); err != nil {
		log.Errorf(ctx, "Failed to log ReplyMessage activity: %v", err)
	}

	log.Infof(ctx, "Message sent to channel %s", session.GetChannelID())
	return nil
}

func (t *ReplyMessageTool) logActivity(ctx context.Context, args ReplyMessageArgs, session SessionContext) error {
	activity := &models.ChatActivity{
		SessionID: session.GetSessionID(),
		ChannelID: session.GetChannelID(),
		Action:    models.ActivityReplyMessage,
		Data:      args,
	}

	return t.activityRepo.Create(ctx, activity)
}