package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// HTTPError hold response error payload.
// Deprecated, please use ResponseError instead.
type HTTPError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("code: %d; message: %s", e.Code, e.Message)
}

// HTTPErrorHandler return custom http error handler.
// Deprecated, please use ErrorHandler.
func HTTPErrorHandler(log Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if err == nil || c.Response().Committed {
			return
		}

		resp := &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "",
		}

		switch v := err.(type) {
		case *echo.HTTPError:
			resp.Code = v.Code
			resp.Message = fmt.Sprint(v.Message)
		case *HTTPError:
			resp = v
		}

		if err := c.JSON(resp.Code, resp); err != nil {
			log.Errorw("could not response", "code", resp.Code, "response_body", resp)
		}
	}
}

// ErrorHandler return custom http error handler.
func ErrorHandler(log Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if err == nil || c.Response().Committed {
			return
		}

		resp := &ResponseError{
			Status:  http.StatusInternalServerError,
			Success: false,
			Err:     err,
		}

		switch v := err.(type) {
		case *echo.HTTPError:
			resp.Status = v.Code
			resp.ErrorMessage = fmt.Sprint(v.Message)
		case *ResponseError:
			resp = v
		default:
			// detect canceled request error
			if errors.Is(err, context.Canceled) && c.Request().Context().Err() == context.Canceled {
				resp.Status = 499
			}
		}

		if resp.Status == http.StatusNotFound && isNotFoundHandler(c.Handler()) {
			resp.ErrorMessage = "no route matched"
		}

		if err := c.JSON(resp.Status, resp); err != nil {
			log.Errorw("could not response", "code", resp.Status, "response_body", resp)
		}
	}
}
