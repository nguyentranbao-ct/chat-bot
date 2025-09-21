package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/carousell/ct-go/pkg/httputils"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestAutoVersion(t *testing.T) {
	e := echo.New()

	AutoVersioning(e, httputils.WithFallbackVersion("1"))
	e.GET("/v1/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "test v1")
	})

	e.GET("/v2/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "test v2")
	})

	e.PATCH("/v4/test/:user_id/foo", func(c echo.Context) error {
		return c.JSON(http.StatusAccepted, map[string]interface{}{
			"success": true,
		})
	})

	t.Run("test v1", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(echo.HeaderAccept, "text/plain; version=1")
		res := httptest.NewRecorder()
		e.ServeHTTP(res, req)

		assert.Equal(t, "text/plain; charset=utf-8; version=1", res.Result().Header.Get(echo.HeaderContentType))
	})

	t.Run("test defautl version", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(echo.HeaderAccept, "application/json")
		res := httptest.NewRecorder()
		e.ServeHTTP(res, req)

		assert.Equal(t, "application/json; charset=utf-8; version=1", res.Result().Header.Get(echo.HeaderContentType))
	})

	t.Run("test v4", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/test/123/foo?type=4", nil)
		req.Header.Set(echo.HeaderAccept, "application/json; version=4")
		res := httptest.NewRecorder()
		e.ServeHTTP(res, req)

		assert.Equal(t, "application/json; charset=utf-8; version=4", res.Result().Header.Get(echo.HeaderContentType))
		assert.Equal(t, http.StatusAccepted, res.Code)
	})

	t.Run("test not found version", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?type=4", nil)
		req.Header.Set(echo.HeaderAccept, "multipart/form-data; version=3")
		res := httptest.NewRecorder()
		e.ServeHTTP(res, req)

		assert.Equal(t, "multipart/form-data; charset=utf-8; version=3", res.Result().Header.Get(echo.HeaderContentType))
		assert.Equal(t, http.StatusNotFound, res.Code)
	})

	t.Run("test prefixed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v4/test?type=4", nil)
		req.Header.Set(echo.HeaderAccept, "multipart/form-data")
		res := httptest.NewRecorder()
		e.ServeHTTP(res, req)

		assert.Equal(t, "multipart/form-data; charset=utf-8; version=4", res.Result().Header.Get(echo.HeaderContentType))
		assert.Equal(t, http.StatusNotFound, res.Code)
	})
}
