package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestRequestIDMiddleware(t *testing.T) {
	// Create a new Echo instance
	e := echo.New()

	// Define a test handler that returns the request ID from the context
	handler := func(c echo.Context) error {
		reqID, ok := c.Get(XRequestID).(string)
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError, "request ID not found in context")
		}
		ctx := c.Request().Context()
		assert.Equal(t, reqID, GetRequestIDFromContext(ctx))
		assert.Equal(t, reqID, GetRequestIDFromEchoContext(c))

		return c.String(http.StatusOK, reqID)
	}

	// Create a new request with a custom request ID header
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(XRequestID, "custom-request-id")

	// Create a new response recorder
	rec := httptest.NewRecorder()

	// Create a new middleware instance with default configuration
	middleware := RequestID()

	// Invoke the middleware with the test handler
	c := e.NewContext(req, rec)
	err := middleware(handler)(c)

	// Assert that the middleware injected the request ID into the context
	assert.NoError(t, err)
	assert.Equal(t, "custom-request-id", c.Get(XRequestID))

	// Assert that the handler returned the correct response
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "custom-request-id", rec.Body.String())
	assert.Equal(t, "custom-request-id", rec.Header().Get(XRequestID))
}
