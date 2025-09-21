package middleware

import (
	"fmt"

	"github.com/labstack/echo/v4"
)

var (
	DefaultSkipper = func(c echo.Context) bool {
		return false
	}
)

type Skipper func(c echo.Context) bool

type Logger interface {
	Debugf(template string, args ...interface{})
	Infof(template string, args ...interface{})
	Warnf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
	Debugw(template string, args ...interface{})
	Infow(template string, args ...interface{})
	Warnw(template string, args ...interface{})
	Errorw(template string, args ...interface{})
}

type Response struct {
	Status       int         `json:"-"`
	Success      bool        `json:"success"`
	Data         interface{} `json:"data,omitempty"`
	ErrorCode    string      `json:"error_code,omitempty"`
	ErrorMessage string      `json:"error_message,omitempty"`
	ErrorData    interface{} `json:"error_data,omitempty"`
}

type ResponseError struct {
	Status       int         `json:"-"`
	Err          error       `json:"-"`
	Success      bool        `json:"success"`
	ErrorCode    string      `json:"error_code,omitempty"`
	ErrorMessage string      `json:"error_message,omitempty"`
	ErrorData    interface{} `json:"error_data,omitempty"`
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("status: %d, code: %s; message: %+v", e.Status, e.ErrorCode, e.Err)
}
