package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/usecase"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Controller interface {
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

	// Profile endpoints
	GetPartnerAttributes(c echo.Context) error
	UpdatePartnerAttributes(c echo.Context) error
}

type controller struct {
	userUsecase usecase.UserUsecase
}

func NewHandler(userUsecase usecase.UserUsecase) Controller {
	return &controller{
		userUsecase: userUsecase,
	}
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

// Profile endpoints

func (h *controller) GetPartnerAttributes(c echo.Context) error {
	// Get user from context (set by auth middleware)
	user, ok := c.Get("user").(*models.User)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not found in context")
	}

	ctx := c.Request().Context()
	attrs, err := h.userUsecase.GetPartnerAttributes(ctx, user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, attrs)
}

func (h *controller) UpdatePartnerAttributes(c echo.Context) error {
	// Get user from context (set by auth middleware)
	user, ok := c.Get("user").(*models.User)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not found in context")
	}

	var req models.PartnerAttributesRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ctx := c.Request().Context()
	if err := h.userUsecase.UpdatePartnerAttributes(ctx, user.ID, &req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "partner attributes updated successfully",
	})
}
