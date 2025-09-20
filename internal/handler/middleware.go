package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func SetupMiddleware(e *echo.Echo) {
	e.HTTPErrorHandler = errorHandler()

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: func(c echo.Context) bool {
			uri := c.Request().RequestURI
			return uri == "/health" || uri == "/metrics"
		},
	}))
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())

	e.Use(validateHeaders())
}

func errorHandler() echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		var he *echo.HTTPError
		if errors.As(err, &he) {
			c.Logger().Error(err)
		} else {
			he = &echo.HTTPError{
				Code:    http.StatusInternalServerError,
				Message: http.StatusText(http.StatusInternalServerError),
			}
		}

		if !c.Response().Committed {
			if c.Request().Method == http.MethodHead {
				err = c.NoContent(he.Code)
			} else {
				err = c.JSON(he.Code, he)
			}
			if err != nil {
				c.Logger().Error(err)
			}
		}
	}
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
