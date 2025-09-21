package middleware

import (
	"context"
	"net/http"

	httpclient "github.com/carousell/ct-go/pkg/httpclient"
	"github.com/labstack/echo/v4"
)

const (
	XRequestID     = "x-request-id"
	XCorrelationID = "x-correlation-id"
)

func GetRequestID(c echo.Context) string {
	if id := GetRequestIDFromEchoContext(c); id != "" {
		return id
	}
	if id := GetRequestIDFromContext(c.Request().Context()); id != "" {
		return id
	}
	if id := GetRequestIDFromHeader(c.Request().Header); id != "" {
		return id
	}
	return ""
}

func GetRequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(XCorrelationID).(string); ok {
		return id
	}
	if id, ok := ctx.Value(XRequestID).(string); ok {
		return id
	}
	return ""
}

func GetRequestIDFromEchoContext(c echo.Context) string {
	if id, ok := c.Get(XRequestID).(string); ok {
		return id
	}
	if id, ok := c.Get(XCorrelationID).(string); ok {
		return id
	}
	return ""
}

func GetRequestIDFromHeader(h http.Header) string {
	if id := h.Get(XRequestID); id != "" {
		return id
	}
	if id := h.Get(XCorrelationID); id != "" {
		return id
	}
	return ""
}

func InjectRequestID(c echo.Context, reqID string) {
	ctx := c.Request().Context()
	//lint:ignore SA1029 we want to expose this key
	ctx = context.WithValue(ctx, XRequestID, reqID)
	//lint:ignore SA1029 we want to expose this key
	ctx = context.WithValue(ctx, XCorrelationID, reqID)

	c.SetRequest(c.Request().WithContext(ctx))
	c.Set(XRequestID, reqID)
	c.Set(XCorrelationID, reqID)
}

func GenerateRequestID() string {
	return httpclient.GenerateCorrelationID()
}

type (
	RequestIDConfig struct {
		Skipper      Skipper
		GenerateFunc func() string
		DetectFunc   func(echo.Context) string
		InjectFunc   func(echo.Context, string)
	}
)

// DefaultBodyLimitConfig is the default Gzip middleware config.
var DefaultRequestIDConfig = RequestIDConfig{
	Skipper:      DefaultSkipper,
	GenerateFunc: GenerateRequestID,
	DetectFunc:   GetRequestID,
	InjectFunc:   InjectRequestID,
}

func RequestID() echo.MiddlewareFunc {
	return RequestIDWithConfig(DefaultRequestIDConfig)
}

func RequestIDWithConfig(config RequestIDConfig) echo.MiddlewareFunc {
	if config.Skipper == nil {
		config.Skipper = DefaultRequestIDConfig.Skipper
	}
	if config.GenerateFunc == nil {
		config.GenerateFunc = DefaultRequestIDConfig.GenerateFunc
	}
	if config.DetectFunc == nil {
		config.DetectFunc = DefaultRequestIDConfig.DetectFunc
	}
	if config.InjectFunc == nil {
		config.InjectFunc = DefaultRequestIDConfig.InjectFunc
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}
			reqID := config.DetectFunc(c)
			if reqID == "" {
				reqID = config.GenerateFunc()
			}
			config.InjectFunc(c, reqID)
			c.Response().Header().Set(XRequestID, reqID)
			return next(c)
		}
	}
}
