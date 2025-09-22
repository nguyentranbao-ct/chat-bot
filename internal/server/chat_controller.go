package server

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/usecase"
)

type ChatController interface {
	GetChannels(c echo.Context) error
	GetChannelMembers(c echo.Context) error
	SendMessage(c echo.Context) error
	GetChannelEvents(c echo.Context) error
	GetChannelMessages(c echo.Context) error
	MarkAsRead(c echo.Context) error
	SetTyping(c echo.Context) error
}

type chatController struct {
	chatUsecase       *usecase.ChatUseCase
	socketBroadcaster usecase.SocketBroadcaster
}

func NewChatController(chatUsecase *usecase.ChatUseCase, socketBroadcaster usecase.SocketBroadcaster) ChatController {
	return &chatController{
		chatUsecase:       chatUsecase,
		socketBroadcaster: socketBroadcaster,
	}
}

func (cc *chatController) GetChannels(c echo.Context) error {
	user := c.Get("user").(*models.User)

	ctx := c.Request().Context()
	channels, err := cc.chatUsecase.GetUserChannels(ctx, user.Email)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, channels)
}

func (cc *chatController) GetChannelMembers(c echo.Context) error {
	channelIDParam := c.Param("id")
	channelID, err := primitive.ObjectIDFromHex(channelIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid channel ID")
	}

	ctx := c.Request().Context()
	members, err := cc.chatUsecase.GetChannelMembers(ctx, channelID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, members)
}

type SendMessageRequest struct {
	Content     string                 `json:"content"`
	MessageType string                 `json:"message_type"`
	Blocks      []models.MessageBlock  `json:"blocks"`
	Metadata    map[string]interface{} `json:"metadata"`
}

func (cc *chatController) SendMessage(c echo.Context) error {
	user := c.Get("user").(*models.User)

	channelIDParam := c.Param("id")
	channelID, err := primitive.ObjectIDFromHex(channelIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid channel ID")
	}

	var req SendMessageRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Content == "" && len(req.Blocks) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "message content or blocks required")
	}

	ctx := c.Request().Context()
	params := usecase.SendMessageParams{
		ChannelID:   channelID,
		SenderID:    user.ID.Hex(),
		Content:     req.Content,
		MessageType: req.MessageType,
		Blocks:      req.Blocks,
		Metadata:    req.Metadata,
	}
	message, err := cc.chatUsecase.SendMessage(ctx, params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Broadcast message to all channel members via socket
	go func() {
		members, err := cc.chatUsecase.GetChannelMembers(ctx, channelID)
		if err != nil {
			fmt.Printf("Failed to get channel members for socket broadcast: %v\n", err)
			return
		}

		userIDs := make([]string, 0, len(members))
		for _, member := range members {
			userIDs = append(userIDs, member.UserID)
		}

		cc.socketBroadcaster.BroadcastMessageToUsers(userIDs, message)
	}()

	// Send confirmation to sender
	cc.socketBroadcaster.BroadcastMessageSent(user.ID.Hex(), message)

	return c.JSON(http.StatusCreated, message)
}

func (cc *chatController) GetChannelEvents(c echo.Context) error {
	channelIDParam := c.Param("id")
	channelID, err := primitive.ObjectIDFromHex(channelIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid channel ID")
	}

	sinceParam := c.QueryParam("since")
	var sinceTime time.Time
	if sinceParam != "" {
		timestamp, err := strconv.ParseInt(sinceParam, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid since timestamp")
		}
		sinceTime = time.Unix(timestamp, 0)
	} else {
		sinceTime = time.Now().Add(-24 * time.Hour) // Default to last 24 hours
	}

	ctx := c.Request().Context()
	events, err := cc.chatUsecase.GetChannelEvents(ctx, channelID, sinceTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, events)
}

func (cc *chatController) GetChannelMessages(c echo.Context) error {
	channelIDParam := c.Param("id")
	channelID, err := primitive.ObjectIDFromHex(channelIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid channel ID")
	}

	limitParam := c.QueryParam("limit")
	limit := 50 // default
	if limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil {
			limit = l
		}
	}

	var beforeID *primitive.ObjectID
	beforeParam := c.QueryParam("before")
	if beforeParam != "" {
		if bid, err := primitive.ObjectIDFromHex(beforeParam); err == nil {
			beforeID = &bid
		}
	}

	ctx := c.Request().Context()
	messages, err := cc.chatUsecase.GetChannelMessages(ctx, channelID, limit, beforeID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, messages)
}

type MarkAsReadRequest struct {
	MessageID string `json:"message_id" validate:"required"`
}

func (cc *chatController) MarkAsRead(c echo.Context) error {
	user := c.Get("user").(*models.User)

	channelIDParam := c.Param("id")
	channelID, err := primitive.ObjectIDFromHex(channelIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid channel ID")
	}

	var req MarkAsReadRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	messageID, err := primitive.ObjectIDFromHex(req.MessageID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid message ID")
	}

	ctx := c.Request().Context()
	if err := cc.chatUsecase.MarkAsRead(ctx, channelID, user.Email, messageID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "marked as read",
	})
}

type SetTypingRequest struct {
	IsTyping bool `json:"is_typing"`
}

func (cc *chatController) SetTyping(c echo.Context) error {
	user := c.Get("user").(*models.User)

	channelIDParam := c.Param("id")
	channelID, err := primitive.ObjectIDFromHex(channelIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid channel ID")
	}

	var req SetTypingRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	ctx := c.Request().Context()
	if err := cc.chatUsecase.SetTyping(ctx, channelID, user.Email, req.IsTyping); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Broadcast typing status to channel members via socket
	go func() {
		members, err := cc.chatUsecase.GetChannelMembers(ctx, channelID)
		if err != nil {
			fmt.Printf("Failed to get channel members for typing broadcast: %v\n", err)
			return
		}

		userIDs := make([]string, 0, len(members))
		for _, member := range members {
			userIDs = append(userIDs, member.UserID)
		}

		cc.socketBroadcaster.BroadcastTypingToUsers(userIDs, channelID.Hex(), user.ID.Hex(), req.IsTyping)
	}()

	return c.JSON(http.StatusOK, map[string]string{
		"status": "success",
	})
}
