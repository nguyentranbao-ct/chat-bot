package middleware

import (
	"net/http"
	"regexp"

	"github.com/labstack/echo/v4"
)

// CORS return echo middleware that handle cors with regexp pattern
func CORS(pattern *regexp.Regexp) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			respHeader := c.Response().Header()
			respHeader.Set("Vary", "Origin")
			origin := c.Request().Header.Get("Origin")
			if origin == "" || !pattern.MatchString(origin) {
				return next(c)
			}
			respHeader.Set("Access-Control-Allow-Origin", origin)
			if c.Request().Method == "OPTIONS" {
				// `*` only may not cover Authorization header in Safari 12
				respHeader.Set("Access-Control-Allow-Headers", "*, Authorization")
				respHeader.Set("Access-Control-Allow-Methods", "OPTIONS, POST, PUT, DELETE, GET, PATCH, HEAD")
				return c.NoContent(http.StatusOK)
			}

			return next(c)
		}
	}
}
