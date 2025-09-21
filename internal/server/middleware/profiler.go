package middleware

import (
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gopkg.in/alexcesaro/statsd.v2"
)

type (
	ProfilerConfig struct {
		// Skipper defines a function to skip middleware.
		Log     Logger
		Skipper middleware.Skipper
		Address string
		Service string
	}
)

// DefaultBodyLimitConfig is the default Gzip middleware config.
var DefaultProfilerConfig = ProfilerConfig{
	Log:     nil,
	Skipper: defaultSkipper,
	Address: ":8125",
	Service: "default",
}

func defaultSkipper(c echo.Context) bool {
	return false
}

func Profiler() echo.MiddlewareFunc {
	return ProfilerWithConfig(DefaultProfilerConfig)
}

// var re = regexp.MustCompile(`\/(\d+)`)
func ProfilerWithConfig(config ProfilerConfig) echo.MiddlewareFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultProfilerConfig.Skipper
	}
	if config.Address == "" {
		config.Address = DefaultProfilerConfig.Address
	}
	if config.Service == "" {
		config.Service = DefaultProfilerConfig.Service
	}

	client, err := statsd.New(statsd.Address(config.Address))
	if err != nil {
		panic(err)
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}

			req := c.Request()
			res := c.Response()
			t := client.NewTiming()
			if err = next(c); err != nil {
				c.Error(err)
			}

			s := strings.ToLower(fmt.Sprintf("response.%s.%s.%s.%d", config.Service, req.Method, c.Path(), res.Status))
			if config.Log != nil {
				config.Log.Debugf(s)
			}
			t.Send(s)

			return
		}
	}
}
