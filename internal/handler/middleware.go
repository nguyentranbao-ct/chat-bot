package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func SetupMiddleware(e *echo.Echo) {
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: func() string {
			return generateRequestID()
		},
	}))

	e.Use(validateHeaders())
}

func validateHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().URL.Path == "/health" {
				return next(c)
			}

			projectUUID := c.Request().Header.Get("x-project-uuid")
			service := c.Request().Header.Get("Service")

			if projectUUID == "" {
				return echo.NewHTTPError(http.StatusBadRequest, "missing x-project-uuid header")
			}

			if service == "" {
				return echo.NewHTTPError(http.StatusBadRequest, "missing Service header")
			}

			return next(c)
		}
	}
}

func generateRequestID() string {
	return "req-" + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[len(charset)/2]
	}
	return string(b)
}