package middleware

import (
	"reflect"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config responsible to configure middleware
type MetricsConfig struct {
	Skipper             func(c echo.Context) bool
	Namespace           string
	Buckets             []float64
	Subsystem           string
	NormalizeHTTPStatus bool
	MetricsPath         string
	NotFoundPath        string
}

const (
	httpRequestsDuration = "request_duration_seconds"
	notFoundPath         = "/not-found"
)

// DefaultConfig has the default instrumentation config
var DefaultMetricsConfig = MetricsConfig{
	Skipper:   DefaultSkipper,
	Namespace: "",
	Subsystem: "",
	Buckets: []float64{
		0.0005,
		0.001, // 1ms
		0.002,
		0.005,
		0.01, // 10ms
		0.02,
		0.05,
		0.1, // 100 ms
		0.2,
		0.5,
		1.0, // 1s
		2.0,
		5.0,
		10.0, // 10s
		15.0,
		20.0,
		30.0,
	},
	NormalizeHTTPStatus: false,
	MetricsPath:         "/metrics",
	NotFoundPath:        "/not-found",
}

func normalizeHTTPStatus(status int) string {
	if status < 200 {
		return "1xx"
	} else if status < 300 {
		return "2xx"
	} else if status < 400 {
		return "3xx"
	} else if status < 500 {
		return "4xx"
	}
	return "5xx"
}

func isNotFoundHandler(handler echo.HandlerFunc) bool {
	return reflect.ValueOf(handler).Pointer() == reflect.ValueOf(echo.NotFoundHandler).Pointer()
}

// Metrics returns an echo middleware with default config for instrumentation.
func Metrics() echo.MiddlewareFunc {
	return MetricsWithConfig(DefaultMetricsConfig)
}

// MetricsWithConfig returns an echo middleware for instrumentation.
func MetricsWithConfig(config MetricsConfig) echo.MiddlewareFunc {
	httpMetrics, err := registerHttpMetrics(config)
	if err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			httpMetrics = are.ExistingCollector.(*prometheus.HistogramVec)
		} else {
			panic(err)
		}
	}

	var promHandler echo.HandlerFunc
	if config.MetricsPath != "" {
		promHandler = echo.WrapHandler(promhttp.Handler())
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			path := c.Path()

			if promHandler != nil && req.RequestURI == config.MetricsPath {
				return promHandler(c)
			}

			if config.Skipper(c) {
				return next(c)
			}

			// to avoid attack high cardinality of 404
			if isNotFoundHandler(c.Handler()) {
				path = notFoundPath
			}

			start := time.Now()

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			status := ""
			if config.NormalizeHTTPStatus {
				status = normalizeHTTPStatus(c.Response().Status)
			} else {
				status = strconv.Itoa(c.Response().Status)
			}

			httpMetrics.WithLabelValues(status, req.Method, path).Observe(time.Since(start).Seconds())

			return err
		}
	}
}

func registerHttpMetrics(config MetricsConfig) (*prometheus.HistogramVec, error) {
	httpMetrics := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: config.Namespace,
		Subsystem: config.Subsystem,
		Name:      httpRequestsDuration,
		Help:      "Spend time by processing a route",
		Buckets:   config.Buckets,
	}, []string{"code", "method", "path"})
	return httpMetrics, prometheus.Register(httpMetrics)
}
