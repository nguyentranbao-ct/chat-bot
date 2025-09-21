package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
)

func makeRequest(e *echo.Echo, path string, rec http.ResponseWriter) {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	e.ServeHTTP(rec, req)
}

func TestPrometheusMiddleware(t *testing.T) {
	clearRegisteredMetrics(t, DefaultMetricsConfig)
	e := echo.New()
	e.Use(Metrics())
	testEchoMetrics(t, e)
}

func TestWrapPrometheusMiddleware(t *testing.T) {
	clearRegisteredMetrics(t, DefaultMetricsConfig)
	e := echo.New()
	e.Use(Metrics())
	testEchoMetrics(t, e)
}

func clearRegisteredMetrics(t *testing.T, conf MetricsConfig) {
	_, err := registerHttpMetrics(conf)
	if err == nil {
		return
	}
	if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
		httpMetrics := are.ExistingCollector.(*prometheus.HistogramVec)
		httpMetrics.Reset()
		return
	}
	t.Errorf("unexpected error %v", err)
}

func testEchoMetrics(t *testing.T, e *echo.Echo) {
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "test")
	})

	e.GET("/test_echo_error", func(c echo.Context) error {
		return c.String(http.StatusInternalServerError, "test")
	})

	errorHandler := func(c echo.Context) error {
		return fmt.Errorf("internal user error")
	}

	e.GET("/test_user_error_1", errorHandler)
	e.GET("/test_user_error_2", errorHandler)

	rec := httptest.NewRecorder()
	for i := 0; i < 100; i++ {
		makeRequest(e, "/test", rec)            // 100
		makeRequest(e, "/test_echo_error", rec) // 100
	}
	for i := 0; i < 96; i++ {
		// new: 96 per each request, old: 500,GET_/404 96*2
		makeRequest(e, "/test_user_error_1", rec)
		makeRequest(e, "/test_user_error_2", rec)
	}
	for i := 0; i < 69; i++ {
		makeRequest(e, "/test_get_notfound", rec) // new 69 old 404,GET_/404
	}

	// request not found
	req := httptest.NewRequest(http.MethodPost, "/test_post_notfound", nil)
	e.ServeHTTP(rec, req)

	makeRequest(e, "/metrics", rec)
	bodyString := rec.Body.String()
	if !strings.Contains(bodyString, `request_duration_seconds_count{code="200",method="GET",path="/test"} 100`) {
		t.Error("GET_/test doesnt show")
	}
	if !strings.Contains(bodyString, `request_duration_seconds_count{code="500",method="GET",path="/test_echo_error"} 100`) {
		t.Error("GET_/test_echo_error doesnt show")
	}

	// // old assert
	// if !strings.Contains(bodyString, `request_duration_seconds_count{code="500",path="/404"} 192`) {
	// 	t.Error("GET_/test_user_error doesnt show")
	// }

	if !strings.Contains(bodyString, `request_duration_seconds_count{code="500",method="GET",path="/test_user_error_1"} 96`) {
		t.Error("GET_/test_user_error doesnt show")
	}

	if !strings.Contains(bodyString, `request_duration_seconds_count{code="500",method="GET",path="/test_user_error_2"} 96`) {
		t.Error("GET_/test_user_error doesnt show")
	}

	if !strings.Contains(bodyString, `request_duration_seconds_count{code="404",method="GET",path="/not-found"} 69`) {
		t.Error("GET_/not-found doesnt show")
	}
	if !strings.Contains(bodyString, `request_duration_seconds_count{code="404",method="POST",path="/not-found"} 1`) {
		t.Error("POST_/not-found doesnt show")
	}

	// ioutil.WriteFile("a.txt", []byte(bodyString), 0644)
	// t.Logf("response %v '%v'", rec.Code, bodyString)
}
