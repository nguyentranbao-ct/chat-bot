package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/usecase"
)

type MessageHandler struct {
	messageUsecase usecase.MessageUsecase
}

func NewMessageHandler(messageUsecase usecase.MessageUsecase) *MessageHandler {
	return &MessageHandler{
		messageUsecase: messageUsecase,
	}
}

func (h *MessageHandler) ProcessMessage(c echo.Context) error {
	var message models.IncomingMessage
	if err := c.Bind(&message); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.Validate(message); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if message.Metadata.LLM.ChatMode == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing chat_mode in metadata.llm")
	}

	ctx := c.Request().Context()
	if err := h.messageUsecase.ProcessMessage(ctx, &message); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "message processed successfully",
	})
}

func (h *MessageHandler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "chat-bot",
	})
}
