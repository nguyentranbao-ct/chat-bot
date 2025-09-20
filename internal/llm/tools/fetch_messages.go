package tools

import (
	"context"
	"fmt"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/nguyentranbao-ct/chat-bot/internal/client"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repository"
)

type FetchMessagesArgs struct {
	Limit    int    `json:"limit"`
	BeforeTs *int64 `json:"before_ts,omitempty"`
}

type FetchMessagesTool struct {
	chatAPIClient client.ChatAPIClient
	activityRepo  repository.ChatActivityRepository
}

func NewFetchMessagesTool(
	chatAPIClient client.ChatAPIClient,
	activityRepo repository.ChatActivityRepository,
) *FetchMessagesTool {
	return &FetchMessagesTool{
		chatAPIClient: chatAPIClient,
		activityRepo:  activityRepo,
	}
}

func (t *FetchMessagesTool) Execute(ctx context.Context, args FetchMessagesArgs, session SessionContext) (*models.MessageHistory, error) {
	args.Limit = clampLimit(args.Limit, 20, 100)

	beforeTs := args.BeforeTs
	if beforeTs == nil {
		beforeTs = session.GetNextMessageTimestamp()
	}

	req := client.MessageHistoryRequest{
		UserID:    session.GetUserID(),
		ChannelID: session.GetChannelID(),
		Limit:     args.Limit,
		BeforeTs:  beforeTs,
	}

	history, err := t.chatAPIClient.GetMessageHistoryWithParams(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	t.updateSessionTimestamp(history, session)

	if err := t.logActivity(ctx, args, session); err != nil {
		log.Errorf(ctx, "Failed to log FetchMessages activity: %v", err)
	}

	log.Infof(ctx, "Fetched %d messages from channel %s", len(history.Messages), session.GetChannelID())
	return history, nil
}

func (t *FetchMessagesTool) updateSessionTimestamp(history *models.MessageHistory, session SessionContext) {
	if len(history.Messages) > 0 {
		oldestMessage := history.Messages[len(history.Messages)-1]
		session.SaveNextMessageTimestamp(oldestMessage.CreatedAt.UnixMilli())
	}
}

func (t *FetchMessagesTool) logActivity(ctx context.Context, args FetchMessagesArgs, session SessionContext) error {
	activity := &models.ChatActivity{
		SessionID: session.GetSessionID(),
		ChannelID: session.GetChannelID(),
		Action:    models.ActivityFetchMessages,
		Data:      args,
	}

	return t.activityRepo.Create(ctx, activity)
}

func clampLimit(current, min, max int) int {
	if current < min {
		return min
	}
	if current > max {
		return max
	}
	return current
}