package tools

import (
	"context"
	"fmt"

	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/nguyentranbao-ct/chat-bot/internal/client"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repository"
)

type TriggerBuyArgs struct {
	ItemName  string `json:"item_name"`
	ItemPrice string `json:"item_price"`
	Intent    string `json:"intent"`
	Message   string `json:"message,omitempty"`
}

type TriggerBuyTool struct {
	chatAPIClient      client.ChatAPIClient
	activityRepo       repository.ChatActivityRepository
	purchaseIntentRepo repository.PurchaseIntentRepository
}

func NewTriggerBuyTool(
	chatAPIClient client.ChatAPIClient,
	activityRepo repository.ChatActivityRepository,
	purchaseIntentRepo repository.PurchaseIntentRepository,
) *TriggerBuyTool {
	return &TriggerBuyTool{
		chatAPIClient:      chatAPIClient,
		activityRepo:       activityRepo,
		purchaseIntentRepo: purchaseIntentRepo,
	}
}

func (t *TriggerBuyTool) Execute(ctx context.Context, args TriggerBuyArgs, session SessionContext) error {
	intent := &models.PurchaseIntent{
		SessionID: session.GetSessionID(),
		ChannelID: session.GetChannelID(),
		UserID:    session.GetUserID(),
		ItemName:  args.ItemName,
		ItemPrice: args.ItemPrice,
		Intent:    args.Intent,
	}

	if err := t.purchaseIntentRepo.Create(ctx, intent); err != nil {
		return fmt.Errorf("failed to create purchase intent: %w", err)
	}

	if args.Message != "" {
		if err := t.sendPurchaseMessage(ctx, args.Message, session); err != nil {
			log.Errorf(ctx, "Failed to send message after TriggerBuy: %v", err)
		} else {
			log.Infof(ctx, "Message sent to channel %s after TriggerBuy", session.GetChannelID())
		}
	}

	if err := t.logActivity(ctx, args, session); err != nil {
		log.Errorf(ctx, "Failed to log TriggerBuy activity: %v", err)
	}

	log.Infof(ctx, "Purchase intent logged: %s wants to buy %s for %s", session.GetUserID(), args.ItemName, args.ItemPrice)
	return nil
}

func (t *TriggerBuyTool) sendPurchaseMessage(ctx context.Context, message string, session SessionContext) error {
	outgoingMessage := &models.OutgoingMessage{
		ChannelID: session.GetChannelID(),
		SenderID:  session.GetSenderID(),
		Message:   `[PURCHASE_INTENT] ` + message,
	}

	return t.chatAPIClient.SendMessage(ctx, outgoingMessage)
}

func (t *TriggerBuyTool) logActivity(ctx context.Context, args TriggerBuyArgs, session SessionContext) error {
	activity := &models.ChatActivity{
		SessionID: session.GetSessionID(),
		ChannelID: session.GetChannelID(),
		Action:    models.ActivityTriggerBuy,
		Data:      args,
	}

	return t.activityRepo.Create(ctx, activity)
}