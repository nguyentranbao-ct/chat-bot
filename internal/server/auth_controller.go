package server

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/usecase"
)

type AuthController interface {
	Login(c echo.Context) error
	GetProfile(c echo.Context) error
	UpdateProfile(c echo.Context) error
	Logout(c echo.Context) error
}

type authController struct {
	authUsecase *usecase.AuthUseCase
}

func NewAuthController(authUsecase *usecase.AuthUseCase) AuthController {
	return &authController{
		authUsecase: authUsecase,
	}
}

func (ac *authController) Login(c echo.Context) error {
	var req models.LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ctx := c.Request().Context()
	userAgent := c.Request().Header.Get("User-Agent")
	ipAddress := c.RealIP()

	response, err := ac.authUsecase.Login(ctx, req, userAgent, ipAddress)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, response)
}

func (ac *authController) GetProfile(c echo.Context) error {
	user := c.Get("user").(*models.User)
	return c.JSON(http.StatusOK, user)
}

func (ac *authController) UpdateProfile(c echo.Context) error {
	user := c.Get("user").(*models.User)

	var req models.ProfileUpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	ctx := c.Request().Context()
	updatedUser, err := ac.authUsecase.UpdateProfile(ctx, user.ID, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, updatedUser)
}

func (ac *authController) Logout(c echo.Context) error {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing authorization header")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid authorization header format")
	}

	ctx := c.Request().Context()
	if err := ac.authUsecase.RevokeToken(ctx, tokenString); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "logged out successfully",
	})
}
