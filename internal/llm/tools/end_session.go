package tools

import (
	"context"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repository"
)

type EndSessionArgs struct{}

type EndSessionTool struct {
	activityRepo repository.ChatActivityRepository
}

func NewEndSessionTool(
	activityRepo repository.ChatActivityRepository,
) *EndSessionTool {
	return &EndSessionTool{
		activityRepo: activityRepo,
	}
}

func (t *EndSessionTool) Execute(ctx context.Context, args EndSessionArgs, session SessionContext) error {
	log.Infof(ctx, "Ending session %s as requested by tool", session.GetSessionID().Hex())

	if err := session.EndSession(); err != nil {
		return err
	}

	if err := t.logActivity(ctx, args, session); err != nil {
		log.Errorf(ctx, "Failed to log EndSession activity: %v", err)
	}

	return nil
}

func (t *EndSessionTool) logActivity(ctx context.Context, args EndSessionArgs, session SessionContext) error {
	activity := &models.ChatActivity{
		SessionID: session.GetSessionID(),
		ChannelID: session.GetChannelID(),
		Action:    models.ActivityEndSession,
		Data:      args,
	}

	return t.activityRepo.Create(ctx, activity)
}