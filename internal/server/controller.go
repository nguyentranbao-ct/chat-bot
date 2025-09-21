package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/usecase"
)

type Controller interface {
	ProcessMessage(c echo.Context) error
	Health(c echo.Context) error
}

type controller struct {
	messageUsecase usecase.MessageUsecase
}

func NewHandler(messageUsecase usecase.MessageUsecase) Controller {
	return &controller{
		messageUsecase: messageUsecase,
	}
}

func (h *controller) ProcessMessage(c echo.Context) error {
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
	if err := h.messageUsecase.ProcessMessage(ctx, message); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "message processed successfully",
	})
}

func (h *controller) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "chat-bot",
	})
}
