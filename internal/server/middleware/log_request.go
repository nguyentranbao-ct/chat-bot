package middleware

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type (
	// LogRequestConfig store middleware configuration
	LogRequestConfig struct {
		Logger       Logger
		Enabled      func(c echo.Context) bool
		RequestID    func(c echo.Context) string
		RequestBody  func(c echo.Context) bool
		ResponseBody func(c echo.Context) bool
		FormValues   func(c echo.Context) bool
		QueryParams  func(c echo.Context) bool
		ParamValues  func(c echo.Context) bool
		KeyAndValues func(c echo.Context) []interface{}
	}
	bodyDumpWriter struct {
		io.Writer
		http.ResponseWriter
	}
)

// LogRequest wraps echo middleware with custom config. All are enabled by default.
func LogRequest(config LogRequestConfig) echo.MiddlewareFunc {
	defFunc := func(c echo.Context) bool {
		return true
	}
	nopFunc := func(c echo.Context) bool {
		return false
	}
	if config.Logger == nil {
		panic("Logger is required to use LogRequest")
	}
	if config.Enabled == nil {
		config.Enabled = defFunc
	}
	if config.RequestBody == nil {
		config.RequestBody = defFunc
	}
	if config.ResponseBody == nil {
		config.ResponseBody = defFunc
	}
	if config.FormValues == nil {
		config.FormValues = defFunc
	}
	if config.QueryParams == nil {
		config.QueryParams = nopFunc
	}
	if config.ParamValues == nil {
		config.ParamValues = nopFunc
	}
	if config.RequestID == nil {
		config.RequestID = GetRequestID
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !config.Enabled(c) {
				return next(c)
			}

			// Request
			start := time.Now()
			req := c.Request()
			res := c.Response()

			// request logging
			logReqBody := config.RequestBody(c)
			logResBody := config.ResponseBody(c)
			logQueryParams := config.QueryParams(c)
			logFormValues := config.FormValues(c)
			logParamValues := config.ParamValues(c)

			var reqBody json.RawMessage
			if logReqBody {
				contentType := req.Header.Get(echo.HeaderContentType)
				if strings.HasPrefix(contentType, echo.MIMEApplicationJSON) {
					reqBody, _ = ioutil.ReadAll(req.Body)
					if len(reqBody) == 0 {
						reqBody = nil
					}
					req.Body = ioutil.NopCloser(bytes.NewBuffer(reqBody))
				}
			}
			var resBuf bytes.Buffer
			if logResBody {
				mw := io.MultiWriter(res.Writer, &resBuf)
				writer := &bodyDumpWriter{Writer: mw, ResponseWriter: res.Writer}
				res.Writer = writer
			}

			err := next(c)
			if err != nil {
				c.Error(err)
			}
			end := time.Since(start)

			message := ""
			args := make([]interface{}, 0, 32)
			args = append(args,
				"status", res.Status,
				"method", req.Method,
				"uri", req.RequestURI,
				"latency_ms", end.Milliseconds(),
				"real_ip", c.RealIP(),
				"user_agent", req.UserAgent(),
			)

			service := req.Header.Get("service")
			if service != "" {
				args = append(args, "service", service)
			}

			userID := GetUserID(c)
			if userID != "" {
				args = append(args, "user_id", userID)
			}

			if logQueryParams {
				query := c.QueryParams()
				if len(query) > 0 {
					args = append(args, "query", c.QueryParams())
				}
			}
			if logFormValues {
				if len(req.Form) > 0 {
					args = append(args, "form", req.Form)
				}
			}
			if logParamValues {
				params := make(map[string]string)
				for _, name := range c.ParamNames() {
					params[name] = c.Param(name)
				}
				if len(params) > 0 {
					args = append(args, "params", params)
				}
			}
			if config.RequestID != nil {
				args = append(args, "request_id", config.RequestID(c))
			}
			if config.KeyAndValues != nil {
				args = append(args, config.KeyAndValues(c)...)
			}
			if logReqBody {
				args = append(args, "request_body", reqBody)
			}
			if logResBody {
				var resBody interface{}
				contentType := res.Header().Get(echo.HeaderContentType)
				if strings.HasPrefix(contentType, echo.MIMEApplicationJSON) {
					resBody = json.RawMessage(resBuf.Bytes())
				}
				args = append(args, "response_body", resBody)
			}

			switch {
			case res.Status >= 500:
				if err != nil {
					args = append(args, "error", err.Error())
				}
				config.Logger.Errorw(message, args...)
			case res.Status >= 400:
				config.Logger.Warnw(message, args...)
			default:
				config.Logger.Infow(message, args...)
			}

			return err
		}
	}
}

func (w *bodyDumpWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *bodyDumpWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *bodyDumpWriter) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

func (w *bodyDumpWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}
