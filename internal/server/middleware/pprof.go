package middleware

import (
	"net/http"
	"net/http/pprof"
	"sync"

	"github.com/labstack/echo/v4"
)

type pprofEchoHandler struct {
	httpHandler       http.Handler
	wrappedHandleFunc echo.HandlerFunc
	once              sync.Once
}

type PprofConfig struct {
	PathPrefix string
}

type pprofHTTPHandler struct {
	serveHTTP func(w http.ResponseWriter, r *http.Request)
}

var DefaultPprofConfig = PprofConfig{
	PathPrefix: "",
}

func (c *pprofHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.serveHTTP(w, r)
}

func (ceh *pprofEchoHandler) Handle(c echo.Context) error {
	ceh.once.Do(func() {
		ceh.wrappedHandleFunc = ceh.mustWrapHandleFunc(c)
	})
	return ceh.wrappedHandleFunc(c)
}

func (ceh *pprofEchoHandler) mustWrapHandleFunc(c echo.Context) echo.HandlerFunc {
	return echo.WrapHandler(ceh.httpHandler)
}

func fromHTTPHandler(httpHandler http.Handler) *pprofEchoHandler {
	return &pprofEchoHandler{httpHandler: httpHandler}
}

func fromHandlerFunc(serveHTTP func(w http.ResponseWriter, r *http.Request)) *pprofEchoHandler {
	return &pprofEchoHandler{httpHandler: &pprofHTTPHandler{serveHTTP: serveHTTP}}
}

func PprofWrap(e *echo.Echo, opts ...PprofConfig) {
	conf := DefaultPprofConfig
	if len(opts) > 0 {
		conf.PathPrefix = opts[0].PathPrefix
	}

	pprofGroup := e.Group(conf.PathPrefix)
	pprofGroup.GET("/debug/pprof/", fromHandlerFunc(pprof.Index).Handle)
	pprofGroup.GET("/debug/pprof/heap", fromHTTPHandler(pprof.Handler("heap")).Handle)
	pprofGroup.GET("/debug/pprof/goroutine", fromHTTPHandler(pprof.Handler("goroutine")).Handle)
	pprofGroup.GET("/debug/pprof/block", fromHTTPHandler(pprof.Handler("block")).Handle)
	pprofGroup.GET("/debug/pprof/threadcreate", fromHTTPHandler(pprof.Handler("threadcreate")).Handle)
	pprofGroup.GET("/debug/pprof/cmdline", fromHandlerFunc(pprof.Cmdline).Handle)
	pprofGroup.GET("/debug/pprof/profile", fromHandlerFunc(pprof.Profile).Handle)
	pprofGroup.GET("/debug/pprof/symbol", fromHandlerFunc(pprof.Symbol).Handle)
	pprofGroup.GET("/debug/pprof/trace", fromHandlerFunc(pprof.Trace).Handle)
}
