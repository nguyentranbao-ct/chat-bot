package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/usecase"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Controller interface {
	ProcessMessage(c echo.Context) error
	Health(c echo.Context) error

	// User management endpoints
	CreateUser(c echo.Context) error
	GetUser(c echo.Context) error
	UpdateUser(c echo.Context) error
	DeleteUser(c echo.Context) error

	// User attributes endpoints
	SetUserAttribute(c echo.Context) error
	GetUserAttributes(c echo.Context) error
	GetUserAttributeByKey(c echo.Context) error
	RemoveUserAttribute(c echo.Context) error
}

type controller struct {
	messageUsecase usecase.MessageUsecase
	userUsecase    usecase.UserUsecase
}

func NewHandler(messageUsecase usecase.MessageUsecase, userUsecase usecase.UserUsecase) Controller {
	return &controller{
		messageUsecase: messageUsecase,
		userUsecase:    userUsecase,
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

// User management endpoints

type CreateUserRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

func (h *controller) CreateUser(c echo.Context) error {
	var req CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ctx := c.Request().Context()
	user, err := h.userUsecase.CreateUser(ctx, req.Name, req.Email)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, user)
}

func (h *controller) GetUser(c echo.Context) error {
	idParam := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user ID")
	}

	ctx := c.Request().Context()
	user, err := h.userUsecase.GetUser(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	return c.JSON(http.StatusOK, user)
}

type UpdateUserRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

func (h *controller) UpdateUser(c echo.Context) error {
	idParam := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user ID")
	}

	var req UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ctx := c.Request().Context()
	user, err := h.userUsecase.GetUser(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	user.Name = req.Name
	user.Email = req.Email

	if err := h.userUsecase.UpdateUser(ctx, user); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, user)
}

func (h *controller) DeleteUser(c echo.Context) error {
	idParam := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user ID")
	}

	ctx := c.Request().Context()
	if err := h.userUsecase.DeleteUser(ctx, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "user deleted successfully",
	})
}

// User attributes endpoints

type SetUserAttributeRequest struct {
	Key   string   `json:"key" validate:"required"`
	Value string   `json:"value" validate:"required"`
	Tags  []string `json:"tags"`
}

func (h *controller) SetUserAttribute(c echo.Context) error {
	idParam := c.Param("id")
	userID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user ID")
	}

	var req SetUserAttributeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ctx := c.Request().Context()
	if err := h.userUsecase.SetUserAttribute(ctx, userID, req.Key, req.Value, req.Tags); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "user attribute set successfully",
	})
}

func (h *controller) GetUserAttributes(c echo.Context) error {
	idParam := c.Param("id")
	userID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user ID")
	}

	ctx := c.Request().Context()
	attrs, err := h.userUsecase.GetUserAttributes(ctx, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, attrs)
}

func (h *controller) GetUserAttributeByKey(c echo.Context) error {
	idParam := c.Param("id")
	userID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user ID")
	}

	key := c.Param("key")
	if key == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "key is required")
	}

	ctx := c.Request().Context()
	attr, err := h.userUsecase.GetUserAttributeByKey(ctx, userID, key)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	return c.JSON(http.StatusOK, attr)
}

func (h *controller) RemoveUserAttribute(c echo.Context) error {
	idParam := c.Param("id")
	userID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user ID")
	}

	key := c.Param("key")
	if key == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "key is required")
	}

	ctx := c.Request().Context()
	if err := h.userUsecase.RemoveUserAttribute(ctx, userID, key); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "user attribute removed successfully",
	})
}
