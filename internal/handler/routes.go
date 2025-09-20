package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/nguyentranbao-ct/chat-bot/pkg/validator"
)

func SetupRoutes(e *echo.Echo, messageHandler *MessageHandler) {
	e.Validator = validator.NewValidator()

	e.GET("/health", messageHandler.Health)

	api := e.Group("/api/v1")
	api.POST("/messages", messageHandler.ProcessMessage)
}
