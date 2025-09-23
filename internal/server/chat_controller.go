package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/repo/internal_api"
	"github.com/nguyentranbao-ct/chat-bot/internal/usecase"
)

type ChatController interface {
	GetRooms(c echo.Context) error
	GetRoomMembers(c echo.Context) error
	SendMessage(c echo.Context) error
	SendInternalMessage(c echo.Context) error
	GetRoomEvents(c echo.Context) error
	GetRoomMessages(c echo.Context) error
	MarkAsRead(c echo.Context) error
}

type chatController struct {
	chatUsecase *usecase.ChatUseCase
}

func NewChatController(chatUsecase *usecase.ChatUseCase) ChatController {
	return &chatController{
		chatUsecase: chatUsecase,
	}
}

func (cc *chatController) GetRooms(c echo.Context) error {
	user := c.Get("user").(*models.User)

	ctx := c.Request().Context()
	roomMembers, err := cc.chatUsecase.GetUserRooms(ctx, user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Convert RoomMembers to client-facing Room objects
	rooms := make([]*models.Room, len(roomMembers))
	for i, roomMember := range roomMembers {
		rooms[i] = roomMember.ToRoom()
	}

	return c.JSON(http.StatusOK, rooms)
}

func (cc *chatController) GetRoomMembers(c echo.Context) error {
	roomIDParam := c.Param("id")
	roomID, err := primitive.ObjectIDFromHex(roomIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid room ID")
	}

	ctx := c.Request().Context()
	members, err := cc.chatUsecase.GetRoomMembersByRoomID(ctx, roomID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Convert to basic member info for clients
	memberInfos := make([]*models.RoomMemberInfo, len(members))
	for i, member := range members {
		memberInfos[i] = member.ToRoomMemberInfo()
	}

	return c.JSON(http.StatusOK, memberInfos)
}

type SendMessageRequest struct {
	Content     string                 `json:"content"`
	MessageType string                 `json:"message_type"`
	Metadata    map[string]interface{} `json:"metadata"`
}

func (cc *chatController) SendMessage(c echo.Context) error {
	user := c.Get("user").(*models.User)

	roomIDParam := c.Param("id")
	roomID, err := primitive.ObjectIDFromHex(roomIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid room ID")
	}

	var req SendMessageRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Content == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "message content required")
	}

	ctx := c.Request().Context()
	params := usecase.SendMessageParams{
		RoomID:   roomID,
		SenderID: user.ID,
		Content:  req.Content,
		Metadata: req.Metadata,
	}
	message, err := cc.chatUsecase.SendMessage(ctx, params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, message)
}

func (cc *chatController) SendInternalMessage(c echo.Context) error {
	var req internal_api.SendMessageRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	roomID, err := primitive.ObjectIDFromHex(req.RoomID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid room ID")
	}

	senderID, err := primitive.ObjectIDFromHex(req.SenderID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid sender ID")
	}

	ctx := c.Request().Context()
	params := usecase.SendMessageParams{
		RoomID:      roomID,
		SenderID:    senderID,
		Content:     req.Content,
		Metadata:    map[string]interface{}{"source": "internal"},
		SkipPartner: req.SkipPartner,
	}

	message, err := cc.chatUsecase.SendMessage(ctx, params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Message sent successfully",
		"id":      message.ID.Hex(),
	})
}

func (cc *chatController) GetRoomEvents(c echo.Context) error {
	roomIDParam := c.Param("id")
	roomID, err := primitive.ObjectIDFromHex(roomIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid room ID")
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
	events, err := cc.chatUsecase.GetRoomEvents(ctx, roomID, sinceTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, events)
}

func (cc *chatController) GetRoomMessages(c echo.Context) error {
	roomIDParam := c.Param("id")
	roomID, err := primitive.ObjectIDFromHex(roomIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid room ID")
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
	messages, err := cc.chatUsecase.GetRoomMessages(ctx, roomID, limit, beforeID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, messages)
}

func (cc *chatController) MarkAsRead(c echo.Context) error {
	user := c.Get("user").(*models.User)

	roomIDParam := c.Param("id")
	roomID, err := primitive.ObjectIDFromHex(roomIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid room ID")
	}

	ctx := c.Request().Context()
	if err := cc.chatUsecase.MarkAsRead(ctx, roomID, user.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "marked as read",
	})
}
