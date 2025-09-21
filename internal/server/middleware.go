package server

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

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
