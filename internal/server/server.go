package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/carousell/ct-go/pkg/logger"
	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nguyentranbao-ct/chat-bot/internal/config"
	pkgmdw "github.com/nguyentranbao-ct/chat-bot/internal/server/middleware"
	"github.com/nguyentranbao-ct/chat-bot/internal/usecase"
	"go.uber.org/fx"
)

func StartServer(
	lc fx.Lifecycle,
	sd fx.Shutdowner,
	conf *config.Config,
	handler Controller,
	authController AuthController,
	chatController ChatController,
	authUsecase *usecase.AuthUseCase,
) {
	e := echo.New()
	e.Validator = pkgmdw.NewValidator()
	e.HTTPErrorHandler = errorHandler()

	logConfig := pkgmdw.LogRequestConfig{
		Logger: logger.MustNamed("http"),
		Enabled: func(c echo.Context) bool {
			uri := c.Request().RequestURI
			return uri != "/health" && uri != "/metrics"
		},
		KeyAndValues: func(c echo.Context) []any {
			args := make([]any, 0, 4)
			if c.Get("user_id") != nil {
				args = append(args, "user_id", c.Get("user_id"))
			}
			if c.Get("project_id") != nil {
				args = append(args, "project_id", c.Get("project_id"))
			}
			return args
		},
	}

	pkgmdw.AutoVersioning(e)
	e.Use(pkgmdw.Metrics())
	e.Use(pkgmdw.RequestID())
	e.Use(pkgmdw.LogRequest(logConfig))
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			log.Errorw(c.Request().Context(), "PANIC RECOVER", "error", err, "stack", string(stack))
			return nil
		},
	}))

	// CORS middleware to allow web frontend access
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.DELETE, echo.OPTIONS},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowCredentials: true,
	}))

	e.GET("/health", handler.Health)

	api := e.Group("/api/v1")
	api.POST("/messages", handler.ProcessMessage)

	// Authentication routes (no auth required)
	api.POST("/auth/login", authController.Login)

	// Protected routes
	authMiddleware := pkgmdw.JWTAuth(authUsecase)

	// User profile routes
	api.GET("/auth/me", authController.GetProfile, authMiddleware)
	api.PUT("/auth/profile", authController.UpdateProfile, authMiddleware)
	api.POST("/auth/logout", authController.Logout, authMiddleware)

	// Chat routes
	chatGroup := api.Group("/chat", authMiddleware)
	chatGroup.GET("/channels", chatController.GetChannels)
	chatGroup.GET("/channels/:id/members", chatController.GetChannelMembers)
	chatGroup.POST("/channels/:id/messages", chatController.SendMessage)
	chatGroup.GET("/channels/:id/messages", chatController.GetChannelMessages)
	chatGroup.GET("/channels/:id/events", chatController.GetChannelEvents)
	chatGroup.POST("/channels/:id/read", chatController.MarkAsRead)
	chatGroup.POST("/channels/:id/typing", chatController.SetTyping)

	// User management routes (keeping for backward compatibility)
	api.POST("/users", handler.CreateUser)
	api.GET("/users/:id", handler.GetUser)
	api.PUT("/users/:id", handler.UpdateUser)
	api.DELETE("/users/:id", handler.DeleteUser)

	// User attributes routes
	api.POST("/users/:id/attributes", handler.SetUserAttribute)
	api.GET("/users/:id/attributes", handler.GetUserAttributes)
	api.GET("/users/:id/attributes/:key", handler.GetUserAttributeByKey)
	api.DELETE("/users/:id/attributes/:key", handler.RemoveUserAttribute)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Infow(ctx, "starting HTTP server", "addr", conf.Server.Addr)
				if err := e.Start(conf.Server.Addr); !errors.Is(err, http.ErrServerClosed) {
					sd.Shutdown()
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return e.Shutdown(ctx)
		},
	})
}
