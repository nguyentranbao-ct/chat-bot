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
	"go.uber.org/fx"
)

func StartServer(
	lc fx.Lifecycle,
	sd fx.Shutdowner,
	conf *config.Config,
	handler Controller,
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

	e.GET("/health", handler.Health)

	api := e.Group("/api/v1")
	api.POST("/messages", handler.ProcessMessage)

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
