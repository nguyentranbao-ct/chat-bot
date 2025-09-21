package middleware

import (
	"github.com/carousell/ct-go/pkg/httputils"
	"github.com/labstack/echo/v4"
)

// AutoVersioning for common api handler
func AutoVersioning(e *echo.Echo, args ...httputils.AutoVersioningOption) {
	versioning := httputils.NewAutoVersioning(args...)
	pre := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			versioning.Handle(c.Response().Writer, c.Request())
			return next(c)
		}
	}

	e.Pre(pre)
}
